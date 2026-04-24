import axios from 'axios';
import { getCredentialsProvider } from '../pages/login/CredentialProvider';


export const apiInstance = axios.create({
    baseURL: import.meta.env.VITE_API_URL,
});

let axosInit = false
let refreshPromise: Promise<boolean> | null = null;

export function setupAxiosInterceptors() {
    if (axosInit) {
        console.log("setupAxiosInterceptors deja appelee")
        return
    }
    // Intercepteur pour injecter le token automatiquement
    apiInstance.interceptors.request.use(async (config) => {
        try {
            // Vérifie si le token expire dans moins de 30s et le rafraîchit si nécessaire.
            // On utilise une promesse partagée pour éviter que plusieurs requêtes simultanées 
            // ne déclenchent plusieurs rafraîchissements en parallèle.
            if (!refreshPromise) {
                refreshPromise = getCredentialsProvider().updateToken(30).finally(() => {
                    refreshPromise = null;
                });
            }
            await refreshPromise;

            const token = getCredentialsProvider().getToken();
            if (token) {
                config.headers.Authorization = `Bearer ${token}`;
            }
        } catch (error) {
            console.error("Erreur lors du rafraîchissement du token", error);
            getCredentialsProvider().signOut();
        }
        return config;
    });

    apiInstance.interceptors.response.use(
        response => response,
        error => {
            if (error.response?.status === 403) {
                getCredentialsProvider().signOut();
            }
            return Promise.reject(error);
        }
    );

    axosInit = true
}