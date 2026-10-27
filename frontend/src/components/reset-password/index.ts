import "../content-card/index.ts";
import { Component, element, staticElement } from "../../custom-elements";

@element("reset-password", "./index.html")
@staticElement
export default class ResetPasswordForm extends Component {
    #token!: string;

    connectedCallback(): void {
        this.#token =
            new URL(window.location.href).searchParams.get("token") ?? "";
        this.#setupEventListeners();
        if (!this.#token) {
            this.select("#error").textContent = "Missing reset link. Please request a new one.";
            this.select<HTMLButtonElement>("#reset-password-submit").disabled = true;
        }
    }

    // --- Events ---
    #setupEventListeners(): void {
        const newPasswordElement =
            this.select<HTMLInputElement>("#new-password");

        const loginForm = this.select<HTMLFormElement>("#reset-password-form");
        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();

            if (!this.#token || this.select<HTMLButtonElement>("#reset-password-submit").disabled) return;
            if (!loginForm.checkValidity()) {
                loginForm.reportValidity();
                return;
            }

            const errorElement = this.select("#error");
            const submitButton = this.select<HTMLButtonElement>(
                "#reset-password-submit",
            );
            errorElement.textContent = "";
            submitButton.disabled = true;

            const newPassword = newPasswordElement.value;

            try {
                await this.#resetPassword(newPassword);
            } catch (error) {
                submitButton.disabled = false;
                if (error instanceof Error) {
                    errorElement.textContent = error.message;
                } else {
                    errorElement.textContent = "Could not reset your password. Please try again.";
                }
                return;
            }

            window.location.assign("/auth/sign-in/?success=reset_password");
        });

        const confirmPasswordElement =
            this.select<HTMLInputElement>("#confirm-password");

        const validateConfirmation = () => {
            if (newPasswordElement.value !== confirmPasswordElement.value) {
                confirmPasswordElement.setCustomValidity(
                    "Passwords do not match",
                );
            } else {
                confirmPasswordElement.setCustomValidity("");
            }
        };
        confirmPasswordElement.addEventListener("input", validateConfirmation);
        newPasswordElement.addEventListener("input", validateConfirmation);
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
            const error = await response.json().catch(() => null);
            throw new Error(response.status === 429
                ? "Too many requests. Please wait a minute and try again."
                : error?.message ?? "Could not reset your password. Please try again.");
        }
    }
}
