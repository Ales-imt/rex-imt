import { useForm, type DefaultValues, type FieldValues } from 'react-hook-form';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { zodResolver } from '@hookform/resolvers/zod';
import { type Datasource, ApiError, ERROR_CODE } from './def';

import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import { useNotifications } from '@toolpad/core/useNotifications';
import { Box, Button } from '@mui/material';
import { useCrudContext } from './CrudContext';


export type FormMode = 'create' | 'show' | 'edit';

interface Props<D extends FieldValues> {
  datasource: Datasource<D>
  initialData: DefaultValues<D>
  mode: FormMode
}

export function Form<D extends FieldValues>({ initialData, mode, datasource, }: Props<D>) {
  const { rootPath } = useCrudContext();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const notifications = useNotifications();
  const isReadOnly = mode === 'show';


  const { register, handleSubmit, control, setError, formState: { errors }, getValues, setValue } = useForm<D>({
    resolver: zodResolver(datasource.schema),
    // READ : On pré-remplit le formulaire avec les données existantes
    defaultValues: initialData,
  });

  const mutation = useMutation({
    // Choix dynamique de la fonction API
    mutationFn: mode === 'edit' ? datasource.update : datasource.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: datasource.queryKey });
      navigate(rootPath);
    },
    onError: (error) => {
      if (error instanceof ApiError) {
        if (error.payload?.code === ERROR_CODE.VALIDATION_ERROR) {
          const details = error.payload.details;
          if (details?.errors) {
            // On parcourt les erreurs renvoyées par le serveur pour les afficher sur les champs
            Object.entries(details.errors).forEach(([field, message]) => {
              setError(field as any, { type: 'server', message: message as string });
            });
          }
          return
        } else if (error.payload?.code === ERROR_CODE.OPTIMISTIC_LOCKING_FAILURE) {
          notifications.show('Modifié par un autre utilisateur. Veuillez réessayer.');
          return
        }
      }
      notifications.show('Une erreur est survenue lors de la soumission du formulaire. Veuillez réessayer.');
    }
  });

  const onSubmit = (data: D) => {
    mutation.mutate(data);
  };

  return (
    <LocalizationProvider dateAdapter={AdapterDayjs}>

      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <form onSubmit={handleSubmit(onSubmit)} style={{ display: 'flex', flexDirection: 'column', gap: '10px', maxWidth: '500px', width: '100%' }}>
          <h2>
            {mode === 'show' ? 'Détails' : mode === 'edit' ? 'Modifier' : 'Ajouter'}
          </h2>

          {/* Utilisation de la fonction/composant extraite */}
          {datasource.render({ register, control, errors, isReadOnly, getValues, setValue })}


          <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mt: 2 }}>
            <Button variant="outlined" onClick={() => navigate(rootPath)}>
              {mode === 'show' ? 'Retour' : 'Annuler'}
            </Button>
            {mode !== 'show' && (
              <Button type="submit" variant="contained" disabled={mutation.isPending}>
                {mutation.isPending ? 'Chargement...' : mode === 'edit' ? 'Mettre à jour' : 'Ajouter'}
              </Button>
            )}
          </Box>

        </form>
      </Box>
    </LocalizationProvider>
  );
}
