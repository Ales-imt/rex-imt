import { useState } from "react";
import {
  Alert,
  Box,
  Button,
  FormControl,
  InputLabel,
  LinearProgress,
  MenuItem,
  Select,
  Stack,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from "@mui/material";
import UploadFileIcon from "@mui/icons-material/UploadFile";
import { apiInstance } from "../../services/api";
import { ENDPOINT_BULLETTIN } from "./def";

// Une colonne et son en-tête (renvoyée par /columns).
interface Colonne {
  letter: string;
  header: string;
}

// État d'un job de génération (renvoyé par /generate/{id}/status).
interface JobStatus {
  state: "running" | "done" | "error";
  done: number;
  total: number;
  error?: string;
}

const attendre = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Extrait le message d'erreur renvoyé par l'API ({"error": "..."}), en gérant
// le cas d'une réponse binaire (responseType blob) où le corps est un Blob.
async function messageErreur(e: unknown): Promise<string> {
  const err = e as { response?: { data?: unknown; statusText?: string }; message?: string };
  const data = err.response?.data;
  if (data instanceof Blob) {
    try {
      const parsed = JSON.parse(await data.text());
      if (parsed?.error) return parsed.error;
    } catch {
      /* corps non JSON */
    }
  } else if (data && typeof data === "object" && "error" in data) {
    return String((data as { error: unknown }).error);
  }
  return err.message ?? err.response?.statusText ?? "Erreur inconnue";
}

// Envoie un FormData et renvoie la donnée JSON de la réponse (authentifiée via
// l'intercepteur axios qui injecte le Bearer token).
async function postForm<T>(url: string, form: FormData): Promise<T> {
  const res = await apiInstance.post<T>(url, form);
  return res.data;
}

export function Bullettin() {
  const [xlsxFile, setXlsxFile] = useState<File | null>(null);
  const [sheets, setSheets] = useState<string[]>([]);
  const [sheet, setSheet] = useState("");
  const [columns, setColumns] = useState<Colonne[]>([]);
  const [filterCol, setFilterCol] = useState(""); // "" = aucun filtre
  const [templateFile, setTemplateFile] = useState<File | null>(null);
  const [format, setFormat] = useState<"pdf" | "docx">("pdf");

  const [status, setStatus] = useState("Prêt.");
  const [isError, setIsError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState<{ done: number; total: number } | null>(null);

  function setMessage(msg: string, error = false) {
    setStatus(msg);
    setIsError(error);
  }

  // Charge la liste des colonnes (avec en-tête) pour la feuille donnée.
  async function chargerColonnes(fichier: File, feuille: string) {
    try {
      const form = new FormData();
      form.append("xlsx", fichier);
      form.append("sheet", feuille);
      const cols = await postForm<Colonne[]>(`${ENDPOINT_BULLETTIN}/columns`, form);
      setColumns(cols);
    } catch (e) {
      setColumns([]);
      setMessage("Erreur (colonnes) : " + (await messageErreur(e)), true);
    }
  }

  async function choisirXlsx(e: React.ChangeEvent<HTMLInputElement>) {
    const fichier = e.target.files?.[0] ?? null;
    setXlsxFile(fichier);
    setSheets([]);
    setSheet("");
    setColumns([]);
    setFilterCol("");
    if (!fichier) return;
    try {
      const form = new FormData();
      form.append("xlsx", fichier);
      const noms = await postForm<string[]>(`${ENDPOINT_BULLETTIN}/sheets`, form);
      setSheets(noms);
      const premiere = noms[0] ?? "";
      setSheet(premiere);
      setMessage(`${noms.length} feuille(s) trouvée(s).`);
      if (premiere) await chargerColonnes(fichier, premiere);
    } catch (e) {
      setMessage("Erreur : " + (await messageErreur(e)), true);
    }
  }

  async function changerFeuille(nouvelleFeuille: string) {
    setSheet(nouvelleFeuille);
    setFilterCol(""); // on réinitialise le filtre au changement de feuille
    if (xlsxFile && nouvelleFeuille) await chargerColonnes(xlsxFile, nouvelleFeuille);
  }

  function choisirTemplate(e: React.ChangeEvent<HTMLInputElement>) {
    setTemplateFile(e.target.files?.[0] ?? null);
  }

  async function generer() {
    if (!xlsxFile || !sheet || !templateFile) {
      setMessage("Renseignez tous les champs avant de générer.", true);
      return;
    }
    setBusy(true);
    setProgress(null);
    setMessage("Démarrage…");
    try {
      // 1. Démarrer le job (réponse immédiate avec l'identifiant).
      const form = new FormData();
      form.append("xlsx", xlsxFile);
      form.append("template", templateFile);
      form.append("sheet", sheet);
      form.append("filterCol", filterCol);
      form.append("format", format);
      const { id } = await postForm<{ id: string }>(`${ENDPOINT_BULLETTIN}/generate`, form);

      // 2. Suivre l'avancement jusqu'à la fin.
      let job: JobStatus;
      for (;;) {
        const res = await apiInstance.get<JobStatus>(`${ENDPOINT_BULLETTIN}/generate/${id}/status`);
        job = res.data;
        setProgress({ done: job.done, total: job.total });
        setMessage(`Génération en cours… ${job.done} / ${job.total}`);
        if (job.state !== "running") break;
        await attendre(1000);
      }
      if (job.state === "error") throw new Error(job.error ?? "Échec de la génération");

      // 3. Télécharger le zip.
      const res = await apiInstance.get(`${ENDPOINT_BULLETTIN}/generate/${id}/download`, {
        responseType: "blob",
      });
      const url = URL.createObjectURL(res.data as Blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `bulletins-${format}.zip`;
      a.click();
      URL.revokeObjectURL(url);

      setMessage(`Terminé : ${job.total} bulletin(s) généré(s).`);
    } catch (e) {
      setMessage("Erreur : " + (await messageErreur(e)), true);
    } finally {
      setBusy(false);
      setProgress(null);
    }
  }

  return (
    <Box sx={{ maxWidth: 640, mx: "auto", p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Génération des bulletins
      </Typography>

      <Stack spacing={3}>
        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Fichier de données (.xlsx)
          </Typography>
          <Stack direction="row" spacing={2} alignItems="center">
            <Button variant="outlined" component="label" startIcon={<UploadFileIcon />}>
              Choisir un fichier
              <input type="file" accept=".xlsx" hidden onChange={choisirXlsx} />
            </Button>
            <Typography variant="body2" color="text.secondary" noWrap>
              {xlsxFile?.name ?? "Aucun fichier"}
            </Typography>
          </Stack>
        </Box>

        <FormControl fullWidth disabled={sheets.length === 0}>
          <InputLabel id="sheet-label">Feuille de données</InputLabel>
          <Select
            labelId="sheet-label"
            label="Feuille de données"
            value={sheet}
            onChange={(e) => changerFeuille(e.target.value)}
          >
            {sheets.map((nom) => (
              <MenuItem key={nom} value={nom}>
                {nom}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <FormControl fullWidth disabled={columns.length === 0}>
          <InputLabel id="filter-label">
            Colonne de filtrage (générer si valeur ≠ « - »)
          </InputLabel>
          <Select
            labelId="filter-label"
            label="Colonne de filtrage (générer si valeur ≠ « - »)"
            value={filterCol}
            onChange={(e) => setFilterCol(e.target.value)}
          >
            <MenuItem value="">Aucun filtre (générer tous les élèves)</MenuItem>
            {columns.map((c) => (
              <MenuItem key={c.letter} value={c.letter}>
                {c.letter} — {c.header}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Template du bulletin (.docx) — variable = champ Ctrl+F9 (voir Alt+F9)
          </Typography>
          <Stack direction="row" spacing={2} alignItems="center">
            <Button variant="outlined" component="label" startIcon={<UploadFileIcon />}>
              Choisir un fichier
              <input type="file" accept=".docx" hidden onChange={choisirTemplate} />
            </Button>
            <Typography variant="body2" color="text.secondary" noWrap>
              {templateFile?.name ?? "Aucun fichier"}
            </Typography>
          </Stack>
        </Box>

        <Box>
          <Typography variant="subtitle2" gutterBottom>
            Format de sortie
          </Typography>
          <ToggleButtonGroup
            exclusive
            color="primary"
            value={format}
            onChange={(_, v) => v && setFormat(v)}
          >
            <ToggleButton value="pdf">PDF</ToggleButton>
            <ToggleButton value="docx">Word (.docx)</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        <Button variant="contained" onClick={generer} disabled={busy}>
          Générer les bulletins
        </Button>

        {progress && (
          <Box>
            <LinearProgress
              variant={progress.total ? "determinate" : "indeterminate"}
              value={progress.total ? (progress.done / progress.total) * 100 : 0}
            />
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              {progress.done} / {progress.total} bulletin(s)
            </Typography>
          </Box>
        )}

        <Alert severity={isError ? "error" : "info"}>{status}</Alert>
      </Stack>
    </Box>
  );
}

export default Bullettin;
