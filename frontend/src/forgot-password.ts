import "./components/content-card/index.ts";
import "./components/navbar/index.ts";
import "./styles/forgot-password.css";

const form = document.querySelector<HTMLFormElement>("#forgot-password-form")!;
const email = document.querySelector<HTMLInputElement>("#email")!;
const button = form.querySelector<HTMLButtonElement>("button")!;
const status = document.querySelector<HTMLElement>("#status")!;
const error = document.querySelector<HTMLElement>("#error")!;

form.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (button.disabled || !form.reportValidity()) return;
    button.disabled = true;
    button.textContent = "Sending…";
    error.textContent = "";
    status.textContent = "";
    try {
        const response = await fetch("/api/auth/request-password-reset", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email: email.value.trim() }),
        });
        if (!response.ok) {
            throw new Error(response.status === 429
                ? "Too many requests. Please wait a minute and try again."
                : "Could not send the reset link. Please try again.");
        }
        status.textContent = "If an account exists for this email address, a reset link is on its way. Check your inbox and spam folder. The link expires in 24 hours.";
    } catch (cause) {
        error.textContent = cause instanceof Error && cause.message !== "Failed to fetch"
            ? cause.message : "Network error. Please check your connection and try again.";
    } finally {
        button.disabled = false;
        button.textContent = "Send reset link";
    }
});
