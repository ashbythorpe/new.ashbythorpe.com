library(dplyr)
library(ggplot2)
library(tibble)
library(purrr)
library(tidyr)

data <- tibble(
  x = 0,
  y = 0,
  color = sample.int(5, 1)
)

add_layer <- function(x, size) {
  x_positions <- 0:(size * 2) / 2
  y_positions <- (size * 2):0 / 2

  positions <- tibble(
    x = c(x_positions, x_positions, -x_positions, -x_positions),
    y = c(y_positions, -y_positions, y_positions, -y_positions),
  ) |> distinct()

  positions$color <- sample.int(5, nrow(positions), replace = TRUE)

  bind_rows(x, positions)
}

for (i in 1:10) {
  data <- add_layer(data, i)
}

data$id <- 1:nrow(data)

plot <- data |>
  filter(color != 5) |>
  mutate(
    data = map2(x, y, \(x, y) tibble(
      x = c(x - 0.5, x, x + 0.5, x),
      y = c(y, y + 0.5, y, y - 0.5),
    ))
  ) |>
  select(-x, -y) |>
  unnest(data) |>
  ggplot(aes(x = x, y = y, fill = factor(color), group = id)) +
    geom_polygon() +
    scale_fill_manual(values = c("#dc2e6b", "#f9dc5c", "#6369d1", "#fa8334"), guide = "none") +
    theme_void()

ggsave("index.svg", path = "src/components/diamond", plot = plot)
