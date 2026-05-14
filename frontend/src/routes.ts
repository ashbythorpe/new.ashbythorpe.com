const path_components = window.location.pathname
  .split("/")
  .filter((x) => x != "");

export function postName(): string {
    return path_components.at(-1) as string
}
