import { authRedirect } from "../../auth-utils";
import { Component, element } from "../../custom-elements";

@element("otp-form", "./index.html")
export default class OtpForm extends Component {
    static observedAttributes = ["email"];

    #inputs: HTMLInputElement[] = [];
    #email: string | null = null;

    connectedCallback(): void {
        this.#inputs = Array.from(
            this.selectAll<HTMLInputElement>(".otp-input"),
        );
        this.setupEventListeners();

        const existingCode = new URL(window.location.href).searchParams.get(
            "code",
        );

        if (existingCode !== null) {
            this.#handlePaste(existingCode);
        }

        setTimeout(() => {
            if (this.#inputs[0]) this.#inputs[0].focus();
        }, 100);
    }

    attributeChangedCallback(
        name: string,
        _oldValue: string | null,
        newValue: string | null,
    ): void {
        if (name === "email" && newValue) {
            this.#email = newValue;
            const emailSpan = this.select("#target-email");
            if (emailSpan) emailSpan.textContent = newValue;
            this.#sendCode();
        }
    }

    private setupEventListeners(): void {
        const form = this.select<HTMLFormElement>("#otp-form");
        const resendBtn = this.select<HTMLButtonElement>("#resend-btn");

        this.#inputs.forEach((input, index) => {
            input.addEventListener("input", (e: Event) => {
                const target = e.target as HTMLInputElement;

                target.value = target.value.replace(/[^0-9]/g, "");

                if (target.value !== "") {
                    if (index < this.#inputs.length - 1) {
                        this.#inputs[index + 1].focus();
                    } else {
                        form.requestSubmit();
                    }
                }
            });

            input.addEventListener("keydown", (e: KeyboardEvent) => {
                const target = e.target as HTMLInputElement;

                if (e.key === "Backspace") {
                    if (target.value === "" && index > 0) {
                        this.#inputs[index - 1].focus();
                        this.#inputs[index - 1].value = "";
                    } else {
                        target.value = "";
                    }
                } else if (e.key === "ArrowLeft" && index > 0) {
                    this.#inputs[index - 1].focus();
                } else if (
                    e.key === "ArrowRight" &&
                    index < this.#inputs.length - 1
                ) {
                    this.#inputs[index + 1].focus();
                }
            });

            input.addEventListener("paste", (e: ClipboardEvent) => {
                e.preventDefault();
                const pastedData = e.clipboardData?.getData("text") || "";
                this.#handlePaste(pastedData, index);
            });
        });

        form.addEventListener("submit", async (e) => {
            e.preventDefault();

            // Gather the 6 digits
            const code = this.#inputs.map((input) => input.value).join("");

            if (code.length === this.#inputs.length) {
                const submitBtn = this.select<HTMLButtonElement>("#verify-btn");
                submitBtn.disabled = true;
                submitBtn.textContent = "Verifying...";

                try {
                    await this.#submitCode(code);
                } catch (e) {
                    if (e instanceof Error) {
                        const errorElement = this.select("#error");
                        errorElement.textContent = e.message;
                    }

                    submitBtn.textContent = "Verify Account";
                    this.#clear();
                    return;
                }

                authRedirect();
            }
        });

        resendBtn.addEventListener("click", async () => {
            await this.#sendCode();
        });
    }

    #handlePaste(code: string, index: number = 0) {
        const pastedNumbers = code
            .replace(/[^0-9]/g, "")
            .slice(0, this.#inputs.length);

        if (pastedNumbers) {
            pastedNumbers.split("").forEach((char, i) => {
                if (this.#inputs[index + i]) {
                    this.#inputs[index + i].value = char;
                }
            });

            // Focus the next empty input, or the last one
            const nextFocusIndex = Math.min(
                index + pastedNumbers.length,
                this.#inputs.length - 1,
            );
            this.#inputs[nextFocusIndex].focus();

            const form = this.select<HTMLFormElement>("#otp-form");

            form.submit();
        }
    }

    async #sendCode() {
        let error = false;
        try {
            const result = await fetch("/api/auth/send-verification", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({ email: this.#email }),
            });

            if (!result.ok) {
                error = true;
                console.error("An error occurred");
            }
        } catch (e) {
            console.error(e);
            error = true;
        }

        if (!error) {
            const resendBtn = this.select<HTMLButtonElement>("#resend-btn");
            resendBtn.disabled = true;
            setTimeout(() => {
                resendBtn.disabled = false;
            }, 1000 * 60);
        }
    }

    async #submitCode(code: string) {
        let result;
        try {
            result = await fetch(`/api/auth/verify-account/${code}`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
            });
        } catch (e) {
            throw new Error("Network error");
        }

        if (!result.ok) {
            const body = await result.json();

            throw new Error(body.message);
        }
    }

    #clear() {
        for (const input of this.#inputs) {
            input.value = "";
        }
    }

    // Public method to reset state if verification fails
    public setVerificationFailed(): void {
        const submitBtn = this.select<HTMLButtonElement>("#verify-btn");
        submitBtn.disabled = false;
        submitBtn.textContent = "Verify Account";

        // Clear inputs and refocus the first one
        this.#inputs.forEach((input) => (input.value = ""));
        this.#inputs[0].focus();
    }
}
