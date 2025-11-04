library(ggplot2)
library(dplyr)
library(tibble)

data <- tibble(
  x = rnorm(50) * 2,
  y = rnorm(50),
  color = 1:50,
)

plot <- ggplot(data, aes(x = x, y = y, color = factor(color))) +
  geom_point(shape = "\U1F7C4", size = 20) +
  scale_color_grey(guide = "none") +
  coord_cartesian(ylim = c(min(data$y) - 0.5, max(data$y) + 0.5)) +
  theme_void()

ggsave("index.svg", path = "src/components/stars", plot = plot, width = 10, height = 5)
