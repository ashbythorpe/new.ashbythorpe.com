# Blog heading studies. Native SVG contours keep calligraphic joins smooth.
# The graph has an explicit irregular layout with cycles and coloured edges.
save_blog_art <- function() {
  write_svg <- function(name, body) {
    writeLines(c('<svg xmlns="http://www.w3.org/2000/svg" width="768" height="300" viewBox="0 0 768 300">', body, "</svg>"), file.path(output, paste0(name, ".svg")))
  }
  swash <- "<path fill=\"#6369d1\" d=\"M82 115 C41 74 66 19 115 30 C167 41 153 114 110 133 C65 155 94 219 186 222 C300 226 426 179 559 199 C613 208 648 232 692 215 C734 199 718 157 690 165 C672 170 673 188 687 193 C662 195 653 172 671 155 C707 119 758 163 733 205 C700 261 615 224 556 217 C418 201 301 254 184 239 C83 227 49 165 99 125 C140 94 144 48 110 45 C79 42 63 81 82 115 Z\"/>\n<path fill=\"#dc2e6b\" d=\"M187 229 C294 241 351 264 436 252 C484 245 511 219 494 204 C482 194 469 205 474 216 C452 209 467 180 490 188 C530 202 514 250 464 263 C360 292 286 245 187 229 Z\"/>\n<path fill=\"#fa8334\" d=\"M99 125 C126 109 144 82 145 61 C168 88 145 127 119 139 C105 147 99 157 97 169 C89 152 91 136 99 125 Z\"/>\n<path fill=\"#f9dc5c\" d=\"M572 203 C600 197 620 185 623 170 C636 189 611 210 594 210 Z\"/>"
  wrap <- "<path fill=\"#6369d1\" d=\"M141 187 C77 168 51 106 75 57 C96 15 149 6 175 39 C197 69 178 101 155 101 C171 90 181 65 165 47 C144 22 109 39 99 65 C80 113 110 158 154 172 C222 195 254 219 354 222 C467 226 568 186 625 157 C691 122 704 77 674 57 C659 46 640 56 643 73 C621 53 650 20 677 37 C728 69 702 131 641 167 C557 217 466 247 353 240 C253 234 217 208 141 187 Z\"/>\n<path fill=\"#dc2e6b\" d=\"M115 153 C76 150 44 175 54 201 C65 231 108 224 116 205 C115 228 80 247 54 226 C11 190 51 137 96 138 Z\"/>\n<path fill=\"#fa8334\" d=\"M354 230 C440 236 538 246 595 211 C620 195 631 173 625 157 C647 184 624 218 596 229 C521 260 421 243 354 230 Z\"/>\n<path fill=\"#f9dc5c\" d=\"M97 67 C88 104 100 137 128 153 C90 146 71 101 97 67 Z\"/>"
  write_svg("blog", swash)
  write_svg("blog-swash", swash)
  write_svg("blog-wrap", wrap)
  # Both new studies start with A's exact closed indigo contour.
  backbone <- strsplit(swash, "\n", fixed = TRUE)[[1]][1]
  junctions <- matrix(c(
    73,76, 115,38, 146,77, 121,129, 89,183, 186,231,
    322,222, 446,208, 559,208, 633,224, 692,224,
    731,191, 707,160, 675,174
  ), ncol = 2, byrow = TRUE)
  diamonds <- apply(junctions, 1, function(point) {
    sprintf('<path d="M%.2f %.2f l6 6 l-6 6 l-6 -6 Z" fill="#171717"/>',
            point[1], point[2]-6)
  })
  decorative_edges <- c(
    '<path d="M73 76 C83 92 103 115 121 129" stroke="#f9dc5c"/>',
    '<path d="M121 129 C166 151 167 204 186 231" stroke="#fa8334"/>',
    '<path d="M89 183 C126 158 171 170 186 231" stroke="#dc2e6b"/>',
    '<path d="M186 231 C278 278 389 278 446 208" stroke="#dc2e6b"/>',
    '<path d="M322 222 C364 252 407 257 446 208" stroke="#fa8334"/>',
    '<path d="M559 208 C610 257 677 253 692 224" stroke="#f9dc5c"/>',
    '<path d="M633 224 C657 213 657 188 675 174" stroke="#dc2e6b"/>',
    '<path d="M675 174 C689 193 711 195 731 191" stroke="#fa8334"/>'
  )
  write_svg("blog-swash-graph", c(
    '<g fill="none" stroke-width="4" stroke-linecap="round">',
    decorative_edges, '</g>', backbone, diamonds))

  # Deform every contour's control points by the same smooth downward envelope.
  # The original silhouette, including its changing thickness, remains intact.
  # Layering from the outer warm colours to the indigo centre echoes the violins.
  path <- sub('.* d="([^"]+)".*', '\\1', backbone)
  tokens <- regmatches(path, gregexpr("[A-Za-z]|-?[0-9]+(?:\\.[0-9]+)?", path, perl = TRUE))[[1]]
  numbers <- which(!grepl("^[A-Za-z]$", tokens))
  layered_contour <- function(offset, colour) {
    shifted <- tokens
    for (i in seq(1, length(numbers), by = 2)) {
      x <- as.numeric(tokens[numbers[i]])
      y <- as.numeric(tokens[numbers[i+1]])
      envelope <- sin(pi * max(0, min(1, (x-65)/675)))^1.1
      shifted[numbers[i+1]] <- sprintf("%.3f", y + offset*envelope)
    }
    sprintf('<path fill="%s" d="%s"/>', colour, paste(shifted, collapse = " "))
  }
  offsets <- c(58, 47, 36, 25, 15, 7, 0)
  colours <- c("#f9dc5c", "#fa8334", "#dc2e6b", "#a65bb8",
               "#8460c6", "#7164cd", "#6369d1")
  layered <- mapply(layered_contour, offsets, colours, USE.NAMES = FALSE)
  write_svg("blog-layered-curves", layered)

  # Equal-width open curves follow the centreline of A. A wider fan beneath the
  # title narrows through the curls; staggered round ends keep the terminals open.
  controls <- list(
    c(82,115, 48,66, 80,25, 116,38),
    c(116,38, 160,54, 145,100, 113,128),
    c(113,128, 55,179, 99,230, 187,231),
    c(187,231, 310,233, 428,188, 558,207),
    c(558,207, 609,214, 660,244, 699,218),
    c(699,218, 748,185, 716,143, 689,162)
  )
  sample_curve <- function(control) {
    p <- matrix(control, ncol = 2, byrow = TRUE)
    t <- seq(0, 1, length.out = 160)
    outer((1-t)^3, p[1,]) + outer(3*(1-t)^2*t, p[2,]) +
      outer(3*(1-t)*t^2, p[3,]) + outer(t^3, p[4,])
  }
  centre <- do.call(rbind, lapply(seq_along(controls), function(i) {
    points <- sample_curve(controls[[i]])
    if (i > 1) points <- points[-1,,drop = FALSE]
    points
  }))
  tangent <- rbind(centre[2,]-centre[1,],
    centre[3:nrow(centre),]-centre[1:(nrow(centre)-2),],
    centre[nrow(centre),]-centre[nrow(centre)-1,])
  normal <- cbind(-tangent[,2], tangent[,1]) / sqrt(rowSums(tangent^2))
  fan <- 0.45 + 0.55 * sin(pi * pmax(0, pmin(1, (centre[,1]-55)/690)))
  distance <- c(0, cumsum(sqrt(rowSums((centre[-1,]-centre[-nrow(centre),])^2))))
  line_colours <- c("#6369d1", "#7164cd", "#8460c6", "#a65bb8",
                    "#dc2e6b", "#fa8334", "#f9dc5c")
  lines <- vapply(seq_along(line_colours), function(i) {
    points <- centre + normal * ((i-1)*6.4*fan)
    keep <- distance >= (i-1)*2 & distance <= max(distance)-(7-i)*3
    points <- points[keep,,drop = FALSE]
    path <- paste(paste(sprintf("%.3f", points[,1]), sprintf("%.3f", points[,2])), collapse = " L ")
    sprintf('<path d="M%s" fill="none" stroke="%s" stroke-width="3.6" stroke-linecap="round" stroke-linejoin="round"/>',
            path, line_colours[i])
  }, character(1))
  write_svg("blog-line-curves", lines)

  # All seven strokes have the same width, but their spacing is always smaller
  # than that width. Back-to-front painting exposes colour bands without gaps.
  # Spacing closes at both terminals, hiding the rear ends under the front curl.
  sweep <- sin(pi * pmax(0, pmin(1, (centre[,1]-55)/690)))
  vertical <- abs(tangent[,2]) / sqrt(rowSums(tangent^2))
  terminal <- sin(pi * distance/max(distance))^0.7
  spacing <- (0.4 + 1.3*vertical + 6*sweep^1.4) * terminal
  overlapping <- vapply(rev(seq_along(line_colours)), function(i) {
    points <- centre + normal * ((i-1)*spacing)
    path <- paste(paste(sprintf("%.3f", points[,1]), sprintf("%.3f", points[,2])), collapse = " L ")
    sprintf('<path d="M%s" fill="none" stroke="%s" stroke-width="10" stroke-linecap="round" stroke-linejoin="round"/>',
            path, line_colours[i])
  }, character(1))
  write_svg("blog-overlapping-curves", overlapping)

  # Seeded, low-frequency deviations act like fitted curves rather than jitter.
  # Compress any large normal gaps smoothly so the body remains overlapping.
  u <- distance/max(distance)
  set.seed(731)
  phases <- runif(7, 0, 2*pi)
  amplitudes <- runif(7, 13, 22)
  offsets <- vapply(seq_along(line_colours), function(i) {
    deviation <- amplitudes[i] * (sin(2*pi*u + phases[i]) +
      0.42*sin(4*pi*u - phases[i])) * sin(pi*u)^1.8
    # Let selected warm fits cross the indigo envelope at different locations.
    excursions <- c(0, 0, 0, 0, 48, 53, 59)
    centres <- c(0.5, 0.5, 0.5, 0.5, 0.63, 0.43, 0.73)
    deviation <- deviation - excursions[i]*exp(-((u-centres[i])/0.105)^2)
    (i-1)*spacing + deviation - 8*exp(-((u-0.59)/0.20)^2)*terminal
  }, numeric(nrow(centre)))
  offsets <- t(apply(offsets, 1, function(row) {
    order <- order(row)
    sorted <- row[order]
    packed <- c(0, cumsum(8*tanh(diff(sorted)/8)))
    packed <- packed - mean(packed) + mean(row)
    result <- numeric(length(row))
    result[order] <- packed
    result
  }))
  # Smooth packing transitions so crossing curves have no abrupt shoulders.
  kernel <- dnorm(-30:30, sd = 10)
  kernel <- kernel/sum(kernel)
  offsets <- apply(offsets, 2, function(values) {
    padded <- c(rep(values[1], 30), values, rep(tail(values, 1), 30))
    as.numeric(stats::filter(padded, kernel, sides = 2))[30+seq_along(values)]
  })
  fitted_points <- lapply(seq_along(line_colours), function(i) {
    centre + normal*offsets[,i]
  })
  stroke <- function(points, colour, width = 10) {
    path <- paste(paste(sprintf("%.3f", points[,1]), sprintf("%.3f", points[,2])), collapse = " L ")
    sprintf('<path d="M%s" fill="none" stroke="%s" stroke-width="%.1f" stroke-linecap="round" stroke-linejoin="round"/>', path, colour, width)
  }
  fitted <- vapply(rev(seq_along(line_colours)), function(i) {
    stroke(fitted_points[[i]], line_colours[i])
  }, character(1))
  write_svg("blog-fitted-curves", fitted)

  # The second study releases the right curl into seven separate, rounded tips.
  # Different lengths end along an airy diagonal outside the title's clear area.
  tips <- cbind(c(614, 631, 651, 676, 701, 726, 747),
                c(105, 123, 141, 158, 175, 191, 208))
  cutoff <- max(which(u <= 0.87))
  released <- vapply(rev(seq_along(line_colours)), function(i) {
    body <- fitted_points[[i]][seq_len(cutoff),,drop = FALSE]
    start <- body[nrow(body),]
    direction <- body[nrow(body),]-body[nrow(body)-1,]
    direction <- direction/sqrt(sum(direction^2))
    end <- tips[i,]
    tail <- sample_curve(c(start, start+direction*31, end+c(24, 22), end))
    stroke(rbind(body, tail[-1,,drop = FALSE]), line_colours[i])
  }, character(1))
  write_svg("blog-released-curves", released)

  # Thin curves spread through the body and both upper curls, following E’s
  # fuller envelope. Smooth local fans keep the turn shapes free of kinks.
  light_spread <- 0.5 + 0.65*sweep^1.8
  end_fade <- (1-exp(-(u/0.025)^2)) * (1-exp(-((1-u)/0.018)^2))
  curl_spacing <- (3.2*exp(-((u-0.09)/0.075)^2) +
                   4.0*exp(-((u-0.96)/0.065)^2)) * end_fade
  # Close the violet/indigo separation on the descending upper-left curl.
  curl_join <- 2.2*exp(-((u-0.17)/0.08)^2)*end_fade
  light_points <- lapply(seq_along(line_colours), function(i) {
    inset <- if (i >= 3) curl_join else 0
    centre + normal*(offsets[,i]*light_spread + (i-1)*curl_spacing - inset)
  })
  light_fitted <- vapply(rev(seq_along(line_colours)), function(i) {
    # Move the curls inward as whole shapes. Ease the connecting body between
    # them, preserving the curls' proportions, stroke widths and colour spacing.
    bridge <- pmax(0, pmin(1, (u-0.24)/0.62))
    bridge <- bridge^2 * (3-2*bridge)
    points <- light_points[[i]]
    points[,1] <- points[,1] + 46 - 155*bridge
    points[,2] <- points[,2] - 12
    stroke(points, line_colours[i], width = 5)
  }, character(1))
  write_svg("blog-fitted-light", light_fitted)
  light_released <- vapply(rev(seq_along(line_colours)), function(i) {
    body <- light_points[[i]][seq_len(cutoff),,drop = FALSE]
    start <- body[nrow(body),]
    direction <- body[nrow(body),]-body[nrow(body)-1,]
    direction <- direction/sqrt(sum(direction^2))
    end <- tips[i,]
    tail <- sample_curve(c(start, start+direction*31, end+c(24, 22), end))
    stroke(rbind(body, tail[-1,,drop = FALSE]), line_colours[i], width = 5)
  }, character(1))
  write_svg("blog-released-light", light_released)

  # An irregular cyclic network follows the swash's envelope. Explicit positions
  # leave the lettering clear and avoid the repeated forks of a recursive tree.
  vertices <- data.frame(
    x = c(86, 145, 177, 124, 66, 90, 155, 205, 142,
          288, 391, 469, 558, 634, 701, 728, 684, 625, 523),
    y = c(56, 40, 96, 138, 115, 190, 228, 195, 180,
          219, 249, 212, 238, 211, 219, 169, 142, 166, 276)
  )
  # from, to, curvature, palette index. Cycles and shared junctions create a
  # graph, while unequal edge spans retain the movement of a pen flourish.
  links <- matrix(c(
    1,2,-15,3,  2,3,-14,3,  3,4,-8,1,  4,5,5,4,
    5,1,-9,3,   1,4,15,2,  4,9,-8,3,  5,6,10,1,
    6,7,14,3,   9,7,-8,4,  9,8,-8,1,  7,8,8,3,
    6,9,-9,2,   7,10,9,3,  8,10,4,4, 10,12,-5,3,
    10,11,17,1, 11,12,8,4, 11,19,12,1, 19,13,8,3,
    12,13,-6,3, 13,14,9,4, 12,18,-9,2, 14,18,-6,1,
    14,15,11,3, 15,16,-9,3, 16,17,-13,4, 17,18,-9,3,
    18,16,-10,1, 18,13,10,3
  ), ncol = 4, byrow = TRUE)
  edges <- apply(links, 1, function(link) {
    from <- vertices[link[1], ]
    to <- vertices[link[2], ]
    dx <- to$x - from$x
    dy <- to$y - from$y
    distance <- sqrt(dx^2 + dy^2)
    control_x <- (from$x + to$x)/2 - dy/distance * link[3]
    control_y <- (from$y + to$y)/2 + dx/distance * link[3]
    sprintf('<path d="M%.2f %.2f Q%.2f %.2f %.2f %.2f" fill="none" stroke="%s" stroke-width="4.5" stroke-linecap="round"/>',
      from$x, from$y, control_x, control_y, to$x, to$y, palette[link[4]])
  })
  nodes <- vapply(seq_len(nrow(vertices)), function(i) {
    x <- vertices$x[i]
    y <- vertices$y[i]
    radius <- if (i %in% c(4, 7, 12, 18)) 6.5 else 5
    sprintf('<path d="M%.2f %.2f l%.2f %.2f l%.2f %.2f l%.2f %.2f Z" fill="#171717"/>',
      x, y-radius, radius, radius, -radius, radius, -radius, -radius)
  }, character(1))
  write_svg("blog-graph", c(edges, nodes))
}
