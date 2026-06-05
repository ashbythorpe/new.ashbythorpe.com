import { Component, element, staticElement } from "../../custom-elements";
import arrowLeft from "../../icons/arrow-left.svg?raw";
import arrowRight from "../../icons/arrow-right.svg?raw";

@staticElement
@element("blog-pagination", "./index.html")
export default class BlogPagination extends Component {
    static observedAttributes = ["current-page", "total-pages"];

    connectedCallback(): void {
        this.render();
        this.setupEventDelegation();
    }

    attributeChangedCallback(
        _name: string,
        oldValue: string | null,
        newValue: string | null,
    ): void {
        if (oldValue !== newValue) {
            this.render();
        }
    }

    private get currentPage(): number {
        return parseInt(this.getAttribute("current-page") ?? "1", 10);
    }

    private get totalPages(): number {
        return parseInt(this.getAttribute("total-pages") ?? "1", 10);
    }

    private createPageURL(pageNumber: number): string {
        const params = new URLSearchParams(window.location.search);
        params.set("page", pageNumber.toString());
        return `${window.location.pathname}?${params.toString()}`;
    }

    private setupEventDelegation() {
        const container = this.select("#container");

        container.addEventListener("click", (e: MouseEvent) => {
            // Find if we clicked on an anchor tag or inside one (like clicking the SVG path)
            const target = (e.target as HTMLElement).closest(
                "a, pagination-arrow",
            );

            if (target && target.hasAttribute("data-page")) {
                // Middle clicks / Ctrl+clicks should still open natively
                if (e.ctrlKey || e.metaKey || e.button !== 0) return;

                e.preventDefault(); // Stop standard navigation

                const newPage = parseInt(target.getAttribute("data-page")!, 10);

                // Emit event to the rest of your app
                this.dispatchEvent(
                    new CustomEvent("page-change", {
                        bubbles: true,
                        composed: true,
                        detail: { page: newPage },
                    }),
                );
            }
        });
    }

    private createArrow(
        direction: "left" | "right",
        disabled: boolean,
        targetPage: number,
    ): HTMLElement {
        const iconHTML = direction === "left" ? arrowLeft : arrowRight;

        if (disabled) {
            const div = document.createElement("div");
            div.className = "page-item disabled";
            div.innerHTML = iconHTML;
            return div;
        } else {
            const link = document.createElement("a");
            link.className = "page-item";
            link.href = this.createPageURL(targetPage);
            link.setAttribute("data-page", targetPage.toString()); // For event listener
            link.innerHTML = iconHTML;
            return link;
        }
    }

    // --- DOM Generation ---
    private render(): void {
        const container = this.select("#container");
        container.innerHTML = "";

        const current = this.currentPage;
        const total = this.totalPages;

        const leftDisabled = current === 1;
        container.appendChild(
            this.createArrow("left", leftDisabled, current - 1),
        );

        const pageNumbers = this.getPageNumbers(current, total);
        for (const page of pageNumbers) {
            container.appendChild(this.createPageButton(page, current));
        }

        const rightDisabled = current >= total;
        container.appendChild(
            this.createArrow("right", rightDisabled, current + 1),
        );
    }

    private createPageButton(
        page: "..." | number,
        current: number,
    ): HTMLElement {
        if (page === "...") {
            const div = document.createElement("div");
            div.className = "page-item disabled";
            div.textContent = "...";
            return div;
        }

        const pageNum = page as number;
        const active = pageNum === current;

        if (active) {
            const div = document.createElement("div");
            div.className = "page-item active";
            div.textContent = pageNum.toString();
            return div;
        } else {
            const link = document.createElement("a");
            link.className = "page-item";
            link.href = this.createPageURL(pageNum);
            link.setAttribute("data-page", pageNum.toString()); // For event listener
            link.textContent = pageNum.toString();
            return link;
        }
    }

    private getPageNumbers(current: number, total: number): (number | "...")[] {
        if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
        if (current <= 3) return [1, 2, 3, 4, "...", total - 1, total];
        if (current >= total - 2)
            return [1, 2, "...", total - 3, total - 2, total - 1, total];
        return [1, "...", current - 1, current, current + 1, "...", total];
    }
}
