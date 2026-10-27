# Site graphics

Run `pnpm graphics` from `frontend`, or `Rscript frontend/scripts/graphics.R`
from the repository root. Requires R with Cairo support and ggplot2; no other R
packages are needed. The script resolves paths relative to itself.

`graphics.R` writes transparent SVGs into `public/graphics/`. Keep those files
with the source: normal Vite builds and deployments do not need R installed.

- `projects.svg`: intersecting paths with extra-bold strokes and coloured endpoints.
- `projects-tall.svg`: the original crossing routes in a taller, narrower format for the desktop projects layout. Run `Rscript frontend/scripts/graphics.R --projects-only` to regenerate both projects assets.
- `blog.svg`: a close calligraphic swash with solid, tapered contours.
- `sign-in.svg`: four opaque asymmetric density silhouettes sharing one centreline.
- `posts/*.svg`: intricate seeded outward-branching trees for each `.dj` file under `post/`.
- `article.svg`: fallback for posts added since the last graphics run.

To change the family, edit `palette`, `routes()`, `violins()` or `network()`, then regenerate.
The seed is derived from each post's relative URL, so editing a post's text or
adding another post does not change existing graphics. Renaming a post does.
The article template selects the matching SVG at build time, falling back to
`article.svg` when one has not been generated. Rerun Vite after regenerating art
for a previously unmatched post.

The images are decorative (`alt=""`) and sit outside the article body. Their
aspect ratios reserve space while loading; the SVGs have no text, external
resources or animation.

Article artwork scatters 440 nodes inside an asymmetric, undulating envelope,
plus one central node. A narrow central bridge opens into fuller side lobes;
alternating dense clusters and loose stretches vary the spacing. A minimum
separation avoids coincident dots. Prim's algorithm connects
the points as a Euclidean minimum spanning tree, giving short irregular edges
without crossings or cycles. Each URL seeds node placement, size and colour, so
regeneration remains reproducible. Only ggplot2 and base R are needed.
Article nodes use rich rose, gold, violet and apricot midtones in equal
sampling proportions, with fine lavender-grey connections. This palette is local to `network()`;
the other page graphics retain the full site palette.
The blog heading uses `blog-graphics.R`, sourced by the main generator.
Explicit closed Bézier contours avoid cusps from offsetting tight curves.
`blog.svg` and `blog-swash.svg` are the close swash; `blog-wrap.svg` is a taller
wrapping alternative; `blog-graph.svg` is an irregular cyclic network with coloured curved edges
and black diamond nodes. Its explicit layout keeps the title clear, with unequal
edge spans and loops in place of recursive branching. The graph uses base R, without
additional graph packages. Open `/graphics/blog-study.html` to compare the studies
with the actual serif title at the same scale. The live header uses J (`blog-fitted-light.svg`), with the heading positioned
inside the artwork's clear central area.

The sign-in artwork uses explicit Gaussian mixtures with independent left and
right profiles. These are violin-inspired silhouettes, not statistical violin
plots of a shared dataset. Tune centres, spreads, weights and widths in
`violins()` to adjust the composition; no random sampling is involved.

The article body overlaps the lower central opening of the network. The artwork
extends beyond the viewport and is clipped horizontally, while the text stays
in its reading column. The network has one central root.

The latest studies derive directly from A’s closed indigo contour:
- `blog-swash-graph.svg` keeps it as a variable-width backbone, with black diamond
  junctions and additional coloured arcs joining those junctions.
- `blog-layered-curves.svg` layers the same contour under a smooth downward
  envelope. Indigo sits at the centre, with violet, rose, orange and yellow
  exposed underneath; the contour retains its original thickness variation.

`blog-line-curves.svg` (F) follows A’s open centreline using seven constant-width
SVG strokes. Normal offsets open the bundle beneath the title and reduce its
spacing through the curls. Arc-length trimming staggers the endpoints, with
round caps and no tapering. Each stroke has the same 3.6-unit width.

`blog-overlapping-curves.svg` (G) uses seven equal 10-unit SVG strokes, painted
from yellow at the back to indigo at the front. Adjacent paths are separated by
less than their stroke width, so the colours overlap without background gaps.
The spacing varies along the curve to form a swash-like envelope, closing to
zero at the terminals so the rear endpoints disappear under the indigo cap.

`blog-fitted-curves.svg` (H) adds reproducible low-frequency deviations and
unequal warm-colour excursions, letting curves pass above and below each other.
Smoothly compressed offsets keep the body overlapping, and a Gaussian pass
smooths the spacing through crossings. Stroke widths remain constant.
`blog-released-curves.svg` (I) uses the same fitted body and opens the right curl
into individually shaped Bézier tails with different lengths and round caps.

`blog-fitted-light.svg` (J) and `blog-released-light.svg` (K) halve the
stroke width to 5 units. Their normal offsets spread up to 1.15 times the
original across the broad body. Local smooth fans also spread the curves
through the upper-left loop and right curl, matching E’s fuller envelope, then
close at the terminals. The released version keeps the original tip positions.

The study page compares H/J and I/K, with A through G in an expandable section.
The live `/blog/` header uses J (`blog-fitted-light.svg`).
