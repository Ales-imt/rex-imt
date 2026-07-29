import axios from "axios"
import type { Session } from "../../SessionContext"


export interface CredentialContext {
    accessToken: string
    refreshToken: string
}

export interface Credentials {
    success: boolean
    error?: string
    session: Session
}

interface LogingResponse {
    name: string,
    surname: string,
    roles: string[]
    access_token: string
    refresh_token: string
}

interface RefreshResponse {
    accessToken: string
    refreshToken: string
}

type SessionObserver = (session: Session | null) => void

// Le back renvoie en cas d'erreur { code, message, details }. Pour une erreur
// de validation, `message` est générique ("Erreur validation") et le texte utile
// est dans `details.message` (details peut être un objet ou un tableau de
// { path, message }). On privilégie donc ce message détaillé.
function errorMessage(data: any): string | undefined {
    const detail = Array.isArray(data?.details) ? data.details[0] : data?.details
    return detail?.message ?? data?.message
}

export class CredentialsProvider {
    private currentSession: Session | null = null
    private observers: Set<SessionObserver> = new Set()
    private accessToken: string | null = null
    private refreshToken: string | null = null
    private tokenExpiry: number | null = null

    public getToken(): string | null {
        return this.accessToken
    }

    public getSession() {
        return this.currentSession
    }

    private setSession(session: Session | null) {
        this.currentSession = session
        this.persistSession(session)
        this.observers.forEach(cb => cb(session))
    }

    public subscribe(observer: SessionObserver): () => void {
        this.observers.add(observer)
        observer(this.currentSession)
        return () => this.observers.delete(observer)
    }

    private storeTokens(accessToken: string, refreshToken: string) {
        this.accessToken = accessToken
        this.refreshToken = refreshToken
        try {
            const payload = JSON.parse(atob(accessToken.split('.')[1]))
            this.tokenExpiry = payload.exp ?? null
        } catch {
            this.tokenExpiry = null
        }
        localStorage.setItem("rex_tokens", JSON.stringify({ accessToken, refreshToken }))
    }

    private persistSession(session: Session | null) {
        if (session) {
            localStorage.setItem("rex_session", JSON.stringify(session))
        } else {
            localStorage.removeItem("rex_session")
            localStorage.removeItem("rex_tokens")
        }
    }

    public listenStorageChanges() {
        window.addEventListener("storage", (event) => {
            if (event.key === "rex_session" || event.key === "rex_tokens") {
                if (event.newValue === null) {
                    this.accessToken = null
                    this.refreshToken = null
                    this.tokenExpiry = null
                    this.setSession(null)
                } else {
                    if (this.restoreSession()) {
                        window.location.href = "/"
                    }
                }
            }
        })
    }

    public restoreSession(): boolean {
        try {
            const rawSession = localStorage.getItem("rex_session")
            const rawTokens = localStorage.getItem("rex_tokens")
            if (!rawSession || !rawTokens) return false

            const session: Session = JSON.parse(rawSession)
            const { accessToken, refreshToken } = JSON.parse(rawTokens)
            this.storeTokens(accessToken, refreshToken)
            this.setSession(session)
            return true
        } catch {
            console.log("imossible restauration de la session")
            return false
        }
    }

    public async signInWithCredentials(email: string, password: string): Promise<Credentials> {
        try {
            const { data: result } = await axios.post<LogingResponse>("/api/v2/auth/login",
                { identifiant: email, password })


            this.storeTokens(result.access_token, result.refresh_token)
            this.setSession({
                user: {
                    email,
                    name: result.name,
                    roles: result.roles
                }
            })

            return {
                success: true,
                session: this.currentSession!
            }
        } catch (err) {
            const error = axios.isAxiosError(err)
                ? (errorMessage(err.response?.data) ?? err.message)
                : "Erreur inconnue"

            return {
                success: false,
                error,
                session: { user: { email: "", name: "", roles: [] } }
            }
        }
    }

    public signOut() {
        try {
            // ne pas utiliser apiInstance, sinon boucle
            axios.get('/api/v2/auth/logout', { headers: { Authorization: `Bearer ${this.getToken()}` } })
        } catch (err) {
            console.log(err)
        }

        this.accessToken = null
        this.refreshToken = null
        this.tokenExpiry = null
        this.setSession(null)
    }

    public async updateToken(minValidity: number): Promise<boolean> {
        const nowInSeconds = Date.now() / 1000

        if (this.tokenExpiry !== null && this.tokenExpiry - nowInSeconds > minValidity) {
            return false
        }

        const { data } = await axios.post<RefreshResponse>("/api/v2/auth/refresh",
            { refreshToken: this.refreshToken })

        this.storeTokens(data.accessToken, data.refreshToken)
        return true
    }
}

let credentialsProvider: CredentialsProvider | null = null

export function getCredentialsProvider(): CredentialsProvider {
    if (credentialsProvider == null) {
        credentialsProvider = new CredentialsProvider()
        credentialsProvider.restoreSession()
        credentialsProvider.listenStorageChanges()
    }
    return credentialsProvider
}
