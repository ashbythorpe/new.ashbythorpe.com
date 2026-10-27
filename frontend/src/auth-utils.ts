export function authRedirect() {
    const redirectParam =
        new URL(window.location.href).searchParams.get("redirect") ?? "/";

    const redirect = new URL(redirectParam, window.location.origin);

    if (redirect.origin !== window.location.origin) {
        console.error("Invalid redirect");
        window.location.assign("/blog/")
    }

    window.location.assign(redirect);
}

export function setEmail(email: string) {
    try {
        sessionStorage.setItem("email", email);
    } catch (e) {
        console.error(e);
    }
}

export function getEmail(): string | null {
    try {
        return sessionStorage.getItem("email");
    } catch (e) {
        console.error(e);
        return null;
    }
}
