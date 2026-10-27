# Run from any directory: Rscript frontend/scripts/graphics.R
# Only ggplot2 is needed; the SVG device ships with R (with Cairo support).
library(ggplot2)

script_arg <- grep("^--file=", commandArgs(), value = TRUE)
script_dir <- dirname(normalizePath(sub("^--file=", "", script_arg[[1]])))
root <- dirname(script_dir)
output <- file.path(root, "public", "graphics")
dir.create(output, recursive = TRUE, showWarnings = FALSE)

palette <- c("#dc2e6b", "#f9dc5c", "#6369d1", "#fa8334")

# A stable, order-sensitive seed: adding another post never changes existing art.
seed_for <- function(key) {
  value <- 0
  for (byte in utf8ToInt(enc2utf8(key))) value <- (value * 131 + byte) %% 2147483646
  as.integer(value + 1)
}

save_art <- function(plot, name, width = 10, height = 2.6) {
  target <- file.path(output, paste0(name, ".svg"))
  dir.create(dirname(target), recursive = TRUE, showWarnings = FALSE)
  # Transparent, text-free SVGs stay crisp at any display size.
  grDevices::svg(target, width = width, height = height, bg = "transparent")
  print(plot + theme_void() + theme(
    plot.margin = margin(8, 8, 8, 8),
    panel.background = element_rect(fill = "transparent", colour = NA),
    plot.background = element_rect(fill = "transparent", colour = NA)
  ))
  invisible(dev.off())
  message("Generated ", name)
}

# Smooth routes with the original endpoint ordering, now with bolder strokes.
routes <- function(line_width = 3.6, point_size = 6) {
  data <- do.call(rbind, lapply(1:9, function(i) {
    x <- seq(0, 1, length.out = 140)
    start <- (i - 1) / 8
    finish <- ((i * 5) %% 9) / 8
    data.frame(x, y = start + (finish - start) * (3 * x^2 - 2 * x^3),
               route = i, colour = palette[(i - 1) %% 4 + 1])
  }))
  ggplot(data, aes(x, y, group = route, colour = colour)) +
    geom_path(linewidth = line_width, alpha = 1, lineend = "round") +
    geom_point(data = data[data$x %in% c(0, 1), ], size = point_size) +
    scale_colour_identity()
}

# Violin-inspired density silhouettes: separate left/right Gaussian mixtures
# share a centreline, with opaque fills layered in distribution order.
violins <- function() {
  # Explicit Gaussian mixtures make the composition reproducible without
  # random sampling. Independent left/right profiles add gentle asymmetry.
  y <- seq(-2.8, 2.8, length.out = 400)
  widths <- c(1.0, 0.82, 0.54, 0.32)
  centres <- list(c(-0.9, 1.15), c(-1.3, 0.7),
                  c(-0.45, 1.45), c(-1.0, 0.85))
  spreads <- list(c(1.0, 0.85), c(0.7, 0.85),
                  c(0.9, 0.55), c(0.6, 0.8))
  weights <- list(c(0.45, 0.55), c(0.5, 0.5),
                  c(0.58, 0.42), c(0.4, 0.6))
  shapes <- do.call(rbind, lapply(1:4, function(i) {
    profile <- function(side) {
      shifts <- if (side == "left") c(-0.2, 0.15) else c(0.25, -0.25)
      # Alternate the lean instead of shifting every layer the same way.
      shifts <- shifts * if (i %% 2 == 0) -1 else 1
      density <- weights[[i]][1] * dnorm(y, centres[[i]][1] + shifts[1], spreads[[i]][1]) +
        weights[[i]][2] * dnorm(y, centres[[i]][2] + shifts[2], spreads[[i]][2])
      # A compact elliptical envelope rounds off the shared ends instead of
      # letting the Gaussian tails stretch into long needles.
      density <- density * sqrt(pmax(0, 1 - (y / 2.8)^2))
      result <- density / max(density) * widths[i]
      result[c(1, length(result))] <- 0
      result
    }
    data.frame(x = c(-profile("left"), rev(profile("right"))),
               y = c(y, rev(y)), distribution = factor(i))
  }))
  ggplot(shapes, aes(x, y, group = distribution, fill = distribution)) +
    geom_polygon(colour = NA) +
    scale_fill_manual(
      # Match the pink-to-periwinkle palette in src/icons/bottom_pattern.svg.
      values = c("#fe89d8", "#da88dc", "#7084e7", "#9385e3"),
      guide = "none"
    ) +
    coord_cartesian(expand = FALSE)
}

# Scatter points inside an undulating, clustered envelope, then compute their Euclidean
# minimum spanning tree. Euclidean MST edges cannot properly cross: replacing
# a crossing with shorter connections would reduce total length.
network <- function(key) {
  set.seed(seed_for(key))
  count <- 440
  nodes <- data.frame(x = 0, y = 0)
  # Uneven lobes and pockets of density break up the arch, while the lower
  # central opening stays clear for the article. Both sides have their own rhythm.
  attempts <- 0
  while (nrow(nodes) <= count && attempts < 30000) {
    attempts <- attempts + 1
    x <- runif(1, -6.2, 6.2)
    envelope <- (0.22 + 0.13 * abs(x)^1.8) *
      (0.78 + 0.22 * sin(2.1 * x + 0.6))
    centre <- -0.065 * x^2 + 0.16 * sin(1.35 * x)
    y <- runif(1, -envelope, envelope) + centre
    # Broad clusters alternate with looser stretches, rather than filling the
    # entire silhouette at a uniform density.
    density <- 0.28 + 0.72 * max(exp(-((x - c(-5.2, -2.8, 2.5, 4.8)) /
      c(0.8, 0.65, 0.85, 0.7))^2))
    separation <- 0.11 + 0.05 * (1 - density)
    if (runif(1) < density &&
        min((nodes$x-x)^2 + (nodes$y-y)^2) > separation^2) {
      nodes <- rbind(nodes, data.frame(x, y))
    }
  }
  # Rich midtones share equal weight, with enough contrast against white.
  network_palette <- c("#bd527a", "#c6a032", "#7771bb", "#d58250")
  nodes$colour <- sample(network_palette, nrow(nodes), replace = TRUE)
  nodes$size <- runif(nrow(nodes), 1.6, 2.8)
  nodes$colour[1] <- network_palette[3]
  nodes$size[1] <- 4.2

  # Prim's algorithm needs no graph package. Seeded point placement makes the
  # layout repeatable; the edge selection minimises Euclidean length.
  distances <- as.matrix(dist(nodes[c("x", "y")]))
  included <- rep(FALSE, nrow(nodes))
  included[1] <- TRUE
  best <- distances[1, ]
  parent <- rep(1L, nrow(nodes))
  segments <- vector("list", nrow(nodes)-1)
  for (i in seq_len(nrow(nodes)-1)) {
    candidates <- best
    candidates[included] <- Inf
    next_node <- which.min(candidates)
    from <- parent[next_node]
    segments[[i]] <- data.frame(x = nodes$x[from], y = nodes$y[from],
      xend = nodes$x[next_node], yend = nodes$y[next_node])
    included[next_node] <- TRUE
    shorter <- !included & distances[next_node, ] < best
    best[shorter] <- distances[next_node, shorter]
    parent[shorter] <- next_node
  }
  segments <- do.call(rbind, segments)
  ggplot() +
    geom_segment(data = segments, aes(x, y, xend = xend, yend = yend),
      colour = "#aaaabd", linewidth = 0.5, alpha = 0.85, lineend = "round") +
    geom_point(data = nodes, aes(x, y, colour = colour, size = size)) +
    scale_colour_identity() + scale_size_identity() +
    coord_cartesian(expand = FALSE, clip = "off")
}

# Heading alternatives use exact Bézier contours and a branching graph.
source(file.path(script_dir, "blog-graphics.R"))

save_art(routes(), "projects", width = 10, height = 2.8)
save_art(routes(line_width = 2.9, point_size = 4.8), "projects-tall", width = 5, height = 5.8)
if ("--projects-only" %in% commandArgs(trailingOnly = TRUE)) quit(save = "no")
save_art(violins(), "sign-in", width = 4.6, height = 6.6)
save_blog_art()
save_art(network("article"), "article", width = 16, height = 5.5)

posts_dir <- file.path(root, "post")
posts <- sort(list.files(posts_dir, pattern = "\\.dj$", recursive = TRUE))
for (post in posts) {
  # Match Vite's URL convention for index.dj and standalone .dj files.
  key <- sub("(^|/)index\\.dj$", "", post)
  key <- sub("\\.dj$", "", key)
  save_art(network(paste0("post/", key)), paste0("posts/", key),
           width = 16, height = 5.5)
}
