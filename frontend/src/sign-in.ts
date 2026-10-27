import "./components/navbar/index.ts";
import "./components/auth-form/index.ts";
import "./styles/sign-in.css";
import { setEmail } from "./auth-utils.ts";

const authForm = document.querySelector("auth-form") as HTMLElement;

authForm.addEventListener("verify", (event) => {
    const email = (event as CustomEvent).detail.email;

    setEmail(email);

    const redirect = new URL(window.location.href).searchParams.get("redirect");
    const url = new URL("/auth/verify-account/", window.location.origin);
    if (redirect !== null) {
        url.searchParams.set("redirect", redirect);
    }
    window.location.assign(url);
});
