import { Component, element, staticElement } from "../../custom-elements";

@element("reset-password", "./index.html")
@staticElement
export default class AuthForm extends Component {
    #token!: string;

    connectedCallback(): void {
        this.#setupEventListeners();
        this.#token =
            new URL(window.location.href).searchParams.get("token") ?? "";
    }

    // --- Events ---
    #setupEventListeners(): void {
        const newPasswordElement =
            this.select<HTMLInputElement>("#new-password");

        const loginForm = this.select<HTMLFormElement>("#reset-password-form");
        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();

            if (!loginForm.checkValidity()) {
                loginForm.reportValidity();
                return;
            }

            const errorElement = this.select("#error");
            const submitButton = this.select<HTMLButtonElement>(
                "#reset-password-submit",
            );
            submitButton.disabled = true;

            const newPassword = newPasswordElement.value;

            try {
                await this.#resetPassword(newPassword);
            } catch (error) {
                submitButton.disabled = false;
                if (error instanceof Error) {
                    errorElement.textContent = error.message;
                    return;
                }
            }

            window.location.assign("/auth/sign-in/?success=reset_password");
        });

        const confirmPasswordElement =
            this.select<HTMLInputElement>("#confirm-password");

        confirmPasswordElement.addEventListener("input", () => {
            if (newPasswordElement.value !== confirmPasswordElement.value) {
                confirmPasswordElement.setCustomValidity(
                    "Passwords do not match",
                );
            } else {
                confirmPasswordElement.setCustomValidity("");
            }
        });
    }

    async #resetPassword(newPassword: string) {
        let response;
        try {
            response = await fetch("/api/auth/reset-password", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    token: this.#token,
                    newPassword,
                }),
            });
        } catch (e) {
            console.error(e);
            throw new Error("Network error");
        }

        if (!response.ok) {
            const error = await response.json();
            console.error(error);

            throw new Error(error.message);
        }
    }
}
