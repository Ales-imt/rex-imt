import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import TextField from '@mui/material/TextField';
import CircularProgress from '@mui/material/CircularProgress';
import Chip from '@mui/material/Chip';
import UploadFileIcon from '@mui/icons-material/UploadFile';
import { useTheme } from '@mui/material/styles';
import { apiInstance } from '../../services/api';
import { ENDPOINT_PRESENCE } from './def';

// ─── Types ────────────────────────────────────────────────────────────────────

type WitnessVerdict =
    | 'CONFORME'
    | 'REECRITURE_DETECTEE'
    | 'CHAINE_CORROMPUE'
    | 'TOKEN_INVALIDE'
    | 'SIGNATURE_INVALIDE';

interface WitnessResult {
    verdict: WitnessVerdict;
    sealedAt?: string;
    coverageSeq?: number;
    brokenSeq?: number;
    tsaName?: string;
    hashHex?: string;
    message: string;
}

interface AnchorRef {
    ledger_seq: number;
    created_at: string;
    tsa_url: string;
}

// ─── Verdicts ─────────────────────────────────────────────────────────────────

const VERDICT_SEVERITY: Record<WitnessVerdict, 'success' | 'error' | 'warning'> = {
    CONFORME: 'success',
    REECRITURE_DETECTEE: 'error',
    CHAINE_CORROMPUE: 'error',
    TOKEN_INVALIDE: 'warning',
    SIGNATURE_INVALIDE: 'warning',
};

const VERDICT_LABEL: Record<WitnessVerdict, string> = {
    CONFORME: 'Conforme',
    REECRITURE_DETECTEE: 'Réécriture détectée',
    CHAINE_CORROMPUE: 'Chaîne corrompue',
    TOKEN_INVALIDE: 'Jeton invalide',
    SIGNATURE_INVALIDE: 'Signature invalide',
};

// Verdicts négatifs sur la chaîne : proposer la dichotomie.
const VERDICT_DICHOTOMY: WitnessVerdict[] = ['REECRITURE_DETECTEE', 'CHAINE_CORROMPUE'];

// ─── Helpers ──────────────────────────────────────────────────────────────────

function formatDate(iso: string | undefined): string {
    if (!iso) return '—';
    return new Date(iso).toLocaleString('fr-FR', { dateStyle: 'long', timeStyle: 'medium' });
}

// Lit un fichier téléversé : un .tsr binaire (DER, premier octet 0x30) est
// converti en base64, un fichier texte (PEM, base64) est collé tel quel.
async function readWitnessFile(file: File): Promise<string> {
    const buf = new Uint8Array(await file.arrayBuffer());
    if (buf.length > 0 && buf[0] === 0x30) {
        let bin = '';
        buf.forEach(b => { bin += String.fromCharCode(b); });
        return btoa(bin);
    }
    return new TextDecoder().decode(buf);
}

// Lit un certificat téléversé : un fichier binaire DER (premier octet 0x30) est
// ré-emballé en PEM, un fichier texte (PEM) est collé tel quel.
async function readCertFile(file: File): Promise<string> {
    const buf = new Uint8Array(await file.arrayBuffer());
    if (buf.length > 0 && buf[0] === 0x30) {
        let bin = '';
        buf.forEach(b => { bin += String.fromCharCode(b); });
        const b64 = btoa(bin).replace(/(.{64})/g, '$1\n').trimEnd();
        return `-----BEGIN CERTIFICATE-----\n${b64}\n-----END CERTIFICATE-----\n`;
    }
    return new TextDecoder().decode(buf);
}

// ─── Main page ────────────────────────────────────────────────────────────────

export function VerifyWitness() {
    const theme = useTheme();

    const [token, setToken] = useState('');
    const [tsaCert, setTsaCert] = useState('');
    const [fileName, setFileName] = useState<string | null>(null);
    const [certFileName, setCertFileName] = useState<string | null>(null);

    // Repères : ancres présentes en base (dates indicatives, pas des preuves).
    const { data: anchors } = useQuery<AnchorRef[]>({
        queryKey: ['ledger-anchors'],
        queryFn: () =>
            apiInstance
                .get<AnchorRef[]>(`${ENDPOINT_PRESENCE}/ledger/anchors`)
                .then(r => r.data),
    });

    const verifyMutation = useMutation<WitnessResult, unknown, void>({
        mutationFn: () =>
            apiInstance
                .post<WitnessResult>(`${ENDPOINT_PRESENCE}/ledger/verify-witness`, {
                    token,
                    tsa_cert: tsaCert,
                })
                .then(r => r.data),
    });

    const handleFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;
        setToken(await readWitnessFile(file));
        setFileName(file.name);
        e.target.value = '';
    };

    const handleCertFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;
        setTsaCert(await readCertFile(file));
        setCertFileName(file.name);
        e.target.value = '';
    };

    const result = verifyMutation.data;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const requestError = (verifyMutation.error as any)?.response?.data?.message
        ?? (verifyMutation.isError ? 'Erreur lors de la vérification.' : null);

    return (
        <Box sx={{ p: 3, maxWidth: 960, mx: 'auto' }}>
            <Typography variant="h5" fontWeight={700} mb={1}>
                Vérification d'un témoin externe
            </Typography>
            <Typography variant="body2" color="text.secondary" mb={2}>
                Collez un témoin d'ancrage (jeton RFC 3161) et obtenez un verdict : la chaîne de
                présences actuelle est-elle conforme au point scellé par ce témoin, ou a-t-elle été
                réécrite ?
            </Typography>

            <Alert severity="info" sx={{ mb: 3 }}>
                <AlertTitle>Provenance du témoin</AlertTitle>
                La preuve repose sur la confrontation de deux sources indépendantes : le témoin
                détenu par un tiers et la base. Utilisez le jeton <strong>reçu par e-mail</strong>{' '}
                depuis la boîte externe (pièce jointe <code>anchor-&lt;seq&gt;.tsr</code>), jamais
                un jeton rechargé depuis la base.
            </Alert>

            {/* Formulaire */}
            <Paper elevation={0} sx={{ p: 3, mb: 3, border: '1px solid', borderColor: 'divider' }}>
                <TextField
                    label="Témoin (jeton RFC 3161) reçu par e-mail"
                    helperText="Collez le contenu en base64 ou PEM, ou téléversez le fichier .tsr ci-dessous"
                    value={token}
                    onChange={e => { setToken(e.target.value); setFileName(null); }}
                    multiline
                    minRows={6}
                    fullWidth
                    slotProps={{ input: { sx: { fontFamily: 'monospace', fontSize: 13 } } }}
                    sx={{ mb: 2 }}
                />

                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
                    <Button variant="outlined" component="label" startIcon={<UploadFileIcon />}>
                        Téléverser le fichier .tsr
                        <input type="file" hidden onChange={handleFile} />
                    </Button>
                    {fileName && <Chip label={fileName} size="small" onDelete={() => { setToken(''); setFileName(null); }} />}
                </Box>

                <TextField
                    label="Certificat TSA (PEM) — optionnel"
                    helperText="Pièce jointe tsa-cert.pem du même e-mail ; à défaut, le certificat racine configuré côté serveur est utilisé"
                    value={tsaCert}
                    onChange={e => { setTsaCert(e.target.value); setCertFileName(null); }}
                    multiline
                    minRows={4}
                    fullWidth
                    slotProps={{ input: { sx: { fontFamily: 'monospace', fontSize: 13 } } }}
                    sx={{ mb: 2 }}
                />

                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
                    <Button variant="outlined" component="label" startIcon={<UploadFileIcon />}>
                        Téléverser le certificat .pem
                        <input type="file" hidden onChange={handleCertFile} />
                    </Button>
                    {certFileName && <Chip label={certFileName} size="small" onDelete={() => { setTsaCert(''); setCertFileName(null); }} />}
                </Box>

                <Button
                    variant="contained"
                    disabled={!token.trim() || verifyMutation.isPending}
                    onClick={() => verifyMutation.mutate()}
                    sx={{ bgcolor: '#4f46e5', '&:hover': { bgcolor: '#4338ca' } }}
                >
                    {verifyMutation.isPending ? <CircularProgress size={18} sx={{ mr: 1, color: '#fff' }} /> : null}
                    Vérifier
                </Button>
            </Paper>

            {requestError && (
                <Alert severity="error" sx={{ mb: 3 }}>{requestError}</Alert>
            )}

            {/* Verdict */}
            {result && (
                <Paper elevation={0} sx={{ mb: 3, border: '1px solid', borderColor: 'divider' }}>
                    <Alert severity={VERDICT_SEVERITY[result.verdict]} sx={{ borderRadius: 0 }}>
                        <AlertTitle>{VERDICT_LABEL[result.verdict] ?? result.verdict} ({result.verdict})</AlertTitle>
                        {result.message}
                    </Alert>
                    <Box sx={{ p: 2 }}>
                        <Table size="small">
                            <TableBody>
                                <TableRow>
                                    <TableCell sx={{ fontWeight: 600, width: 220 }}>Date certifiée (TSA)</TableCell>
                                    <TableCell>{formatDate(result.sealedAt)}</TableCell>
                                </TableRow>
                                {result.tsaName && (
                                    <TableRow>
                                        <TableCell sx={{ fontWeight: 600 }}>Autorité d'horodatage</TableCell>
                                        <TableCell>{result.tsaName}</TableCell>
                                    </TableRow>
                                )}
                                {!!result.coverageSeq && (
                                    <TableRow>
                                        <TableCell sx={{ fontWeight: 600 }}>Couverture (maillon)</TableCell>
                                        <TableCell>{result.coverageSeq}</TableCell>
                                    </TableRow>
                                )}
                                {!!result.brokenSeq && (
                                    <TableRow>
                                        <TableCell sx={{ fontWeight: 600 }}>Rupture au maillon</TableCell>
                                        <TableCell>{result.brokenSeq}</TableCell>
                                    </TableRow>
                                )}
                                {result.hashHex && (
                                    <TableRow>
                                        <TableCell sx={{ fontWeight: 600 }}>Empreinte scellée</TableCell>
                                        <TableCell sx={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>
                                            {result.hashHex}
                                        </TableCell>
                                    </TableRow>
                                )}
                            </TableBody>
                        </Table>
                        {VERDICT_DICHOTOMY.includes(result.verdict) && (
                            <Alert severity="warning" sx={{ mt: 2 }}>
                                <AlertTitle>Localiser l'altération</AlertTitle>
                                Testez un témoin plus ancien : s'il est conforme, l'altération se situe
                                entre sa date certifiée et celle-ci. Répétez en resserrant l'intervalle
                                (dichotomie). La date certifiée affichée à chaque test situe le témoin
                                sur la frise temporelle.
                            </Alert>
                        )}
                    </Box>
                </Paper>
            )}

            {/* Aide */}
            <Paper elevation={0} sx={{ p: 3, mb: 3, border: '1px solid', borderColor: 'divider' }}>
                <Typography variant="subtitle1" fontWeight={700} mb={1}>Comment lire le verdict</Typography>
                <Typography variant="body2" component="ul" color="text.secondary" sx={{ pl: 3, m: 0, '& li': { mb: 0.5 } }}>
                    <li><strong>Conforme</strong> — la chaîne actuelle reproduit l'état scellé : tout ce
                        qui précède le maillon couvert est authentifié.</li>
                    <li><strong>Réécriture détectée</strong> — le hash certifié a disparu de la chaîne :
                        les données ont été modifiées après la date scellée.</li>
                    <li><strong>Chaîne corrompue</strong> — le hash scellé existe encore mais le chaînage
                        interne est rompu avant lui.</li>
                    <li><strong>Jeton / signature invalide</strong> — témoin non exploitable ou non
                        probant : vérifiez le copier-coller et le certificat.</li>
                </Typography>
                <Typography variant="body2" color="text.secondary" mt={1.5}>
                    La garantie du dispositif est la <strong>détectabilité</strong> d'une altération, pas
                    son impossibilité — et elle repose sur des témoins <strong>externes</strong>, détenus
                    par un tiers. La vérification est en lecture seule : aucun témoin collé ici n'est
                    conservé.
                </Typography>
            </Paper>

            {/* Repères : ancres en base */}
            {(anchors?.length ?? 0) > 0 && (
                <>
                    <Typography variant="subtitle1" fontWeight={700} mb={0.5}>
                        Repères : ancres enregistrées en base
                    </Typography>
                    <Typography variant="body2" color="text.secondary" mb={1}>
                        Ces dates proviennent de la base et servent uniquement à situer les intervalles
                        entre témoins ; elles ne constituent pas une preuve.
                    </Typography>
                    <TableContainer component={Paper} elevation={0} sx={{ border: '1px solid', borderColor: 'divider' }}>
                        <Table size="small">
                            <TableHead>
                                <TableRow sx={{ bgcolor: theme.palette.mode === 'dark' ? '#1e293b' : '#f8fafc' }}>
                                    <TableCell>Maillon (seq)</TableCell>
                                    <TableCell>Date d'ancrage</TableCell>
                                    <TableCell>TSA</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {anchors!.map(a => (
                                    <TableRow key={`${a.ledger_seq}-${a.tsa_url}-${a.created_at}`} hover>
                                        <TableCell>{a.ledger_seq}</TableCell>
                                        <TableCell>{formatDate(a.created_at)}</TableCell>
                                        <TableCell>{a.tsa_url}</TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </>
            )}
        </Box>
    );
}
