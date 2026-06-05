export function authRedirect() {
    const redirectParam =
        new URL(window.location.href).searchParams.get("redirect") ?? "/";

    const redirect = new URL(redirectParam, window.location.origin);

    if (redirect.origin !== window.location.origin) {
        throw new Error("Invalid redirect");
    }

    window.location.assign(redirect);
}

export async function sendVerificationEmail(email: string): Promise<boolean> {
    const result = await fetch("/auth/resend-verification", {
        method: "POST",
        headers: {
            "Content-Type": "application/json",
        },
        body: JSON.stringify({ email }),
    });

    if (!result.ok) {
        const body = await result.json();

        if (body.type === "existing-token") {
            return false;
        } else {
            throw new Error(body.message);
        }
    }

    return true;
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
