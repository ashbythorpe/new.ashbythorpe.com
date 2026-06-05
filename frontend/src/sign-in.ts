import "./components/navbar/index.ts";
import "./components/auth-form/index.ts";
import "./styles/global.css";
import { getEmail } from "./auth-utils.ts";

const otpForm = document.querySelector("otp-form") as HTMLElement;

const email = getEmail();
if (email !== null) {
    otpForm.setAttribute("email", email);

    await fetch("/api/auth/send-verification", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ email }),
    });
}
