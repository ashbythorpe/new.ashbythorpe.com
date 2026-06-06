import { authRedirect } from "../../auth-utils";
import { Component, element, staticElement } from "../../custom-elements";

@element("auth-form", "./index.html")
@staticElement
export default class AuthForm extends Component {
    #loginToken: string | null = null;
    #signupToken: string | null = null;

    connectedCallback(): void {
        this.#setupEventListeners();
        this.#setupTurnstiles();

        const authErrorCode = new URL(window.location.href).searchParams.get(
            "auth_error",
        );
        let authError = null;
        switch (authErrorCode) {
            case "bad_verification":
                authError = "The sign in session expired.";
                break;
            case "unverified_email":
                authError =
                    "The primary email of your GitHub account doesn't exist or is unverified.";
                break;
            case "github_server":
                authError = "We couldn't connect to GitHub right now.";
                break;
            case "internal":
                authError = "An internal error occurred.";
                break;
        }

        console.log(authError);
        if (authError !== null) {
            const authErrorElement = this.select("#auth-error");
            authErrorElement.textContent = authError;
        }

        const authSuccessCode = new URL(window.location.href).searchParams.get(
            "success",
        );
        let authSuccess = null;
        switch (authSuccessCode) {
            case "reset_password":
                authSuccess = "Successfully reset password";
                break;
            case "account_created":
                authSuccess = "Account created successfully";
                break;
        }

        if (authSuccess !== null) {
            const authsuccessElement = this.select("#auth-success");
            authsuccessElement.textContent = authSuccess;
        }
    }

    // --- State Management ---
    get isSignup(): boolean {
        return this.internals.states.has("signup");
    }

    set isSignup(value: boolean) {
        if (value) this.internals.states.add("signup");
        else this.internals.states.delete("signup");
    }

    // --- Events ---
    #setupEventListeners(): void {
        // Tab Switching
        this.select("#tab-login").addEventListener(
            "click",
            () => (this.isSignup = false),
        );
        this.select("#tab-signup").addEventListener(
            "click",
            () => (this.isSignup = true),
        );

        const redirect = new URL(window.location.href).searchParams.get(
            "redirect",
        );
        const authLink = this.select<HTMLAnchorElement>("#github-auth-btn");
        const url = new URL("/api/auth/github", window.location.origin);
        if (redirect !== null) {
            url.searchParams.set("redirect", redirect);
        }

        // Don't set href to avoid prefetching
        authLink.addEventListener("click", () => {
            window.location.assign(url);
        });

        const loginForm = this.select<HTMLFormElement>("#login-view");
        loginForm.addEventListener("submit", async (e) => {
            e.preventDefault();

            if (!loginForm.checkValidity()) {
                loginForm.reportValidity();
                return;
            }

            if (!this.#loginToken) {
                return;
            }

            const errorElement = this.select("#login-error");
            const submitButton =
                this.select<HTMLButtonElement>("#login-submit");
            submitButton.disabled = true;

            const email = this.select<HTMLInputElement>("#login-email").value;
            const password =
                this.select<HTMLInputElement>("#login-password").value;

            let verified;
            try {
                verified = await this.#login(email, password);
            } catch (error) {
                this.#resetTurnstiles();
                submitButton.disabled = false;
                if (error instanceof Error) {
                    errorElement.textContent = error.message;
                    return;
                }
            }

            if (verified) {
                authRedirect();
            } else {
                this.#resetTurnstiles();
                this.dispatchEvent(
                    new CustomEvent("verify", {
                        bubbles: true,
                        composed: true,
                        detail: { email },
                    }),
                );
            }
        });

        const signUpPasswordElement =
            this.select<HTMLInputElement>("#signup-password");
        const confirmPasswordElement = this.select<HTMLInputElement>(
            "#signup-confirm-password",
        );

        const signUpForm = this.select<HTMLFormElement>("#signup-view");
        signUpForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            const name = this.select<HTMLInputElement>("#signup-name").value;
            const email = this.select<HTMLInputElement>("#signup-email").value;
            const password = signUpPasswordElement.value;
            const errorElement = this.select("#signup-error");
            const submitButton =
                this.select<HTMLButtonElement>("#signup-submit");
            submitButton.disabled = true;

            if (!signUpForm.checkValidity()) {
                signUpForm.reportValidity();
                return;
            }

            if (!this.#signupToken) {
                return;
            }

            try {
                await this.#signup(email, password, name);
            } catch (error) {
                this.#resetTurnstiles();
                submitButton.disabled = false;
                if (error instanceof Error) {
                    errorElement.textContent = error.message;
                    return;
                }
            }

            this.dispatchEvent(
                new CustomEvent("verify", {
                    bubbles: true,
                    composed: true,
                    detail: { email },
                }),
            );
        });

        confirmPasswordElement.addEventListener("input", () => {
            if (signUpPasswordElement.value !== confirmPasswordElement.value) {
                confirmPasswordElement.setCustomValidity(
                    "Passwords do not match",
                );
            } else {
                confirmPasswordElement.setCustomValidity("");
            }
        });
    }

    #resetTurnstiles() {
        turnstile.reset("#log-in-turnstile");
        turnstile.reset("#sign-up-turnstile");
    }

    #setupTurnstiles() {
        // /* @ts-ignore */
        // if (!window.turnstile) {
        //     console.log("Waiting for Turnstile to load");
        //     setTimeout(() => this.#setupTurnstiles(), 100);
        //     return;
        // }

        // const loginContainer = this.querySelector("#login-turnstile-container");
        // const signupContainer = this.querySelector("#signup-turnstile-container");

        console.log(import.meta.env);

        turnstile.render("#log-in-turnstile", {
            sitekey: import.meta.env.VITE_TURNSTILE_SITE_KEY,
            theme: "auto",
            size: "flexible",
            callback: (token) => {
                this.#loginToken = token;
            },
        });

        // 0x4AAAAAABuPwLoKiglpi_Nz
        turnstile.render("#sign-up-turnstile", {
            sitekey: import.meta.env.VITE_TURNSTILE_SITE_KEY,
            theme: "auto",
            size: "flexible",
            callback: (token) => {
                this.#signupToken = token;
            },
        });
    }

    async #login(email: string, password: string): Promise<boolean> {
        const response = await fetch("/api/auth/login", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                email,
                password,
                turnstileToken: this.#loginToken,
            }),
        });

        if (!response.ok) {
            const error = await response.json();
            console.error(error);

            if (error.type === "not-verified") {
                return false;
            } else {
                throw new Error(error.message);
            }
        }

        return true;
    }

    async #signup(email: string, password: string, name: string) {
        const response = await fetch("/api/auth/sign-up", {
            method: "POST",
            headers: {
                "Content-Type": "application/json",
            },
            body: JSON.stringify({
                email,
                password,
                name,
                turnstileToken: this.#signupToken,
            }),
        });

        if (!response.ok) {
            const error = await response.json();

            throw new Error(error.message);
        }
    }
}
