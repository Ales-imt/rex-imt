import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import type { Datasource } from './def';
import type { FieldValues } from 'react-hook-form';
import { MaterialReactTable, useMaterialReactTable, type MRT_Row, type MRT_TableInstance } from 'material-react-table';
import { Box, Button, darken, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, IconButton, Tooltip, Typography } from '@mui/material';
import AddBoxIcon from '@mui/icons-material/AddBox';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import VisibilityIcon from '@mui/icons-material/Visibility';
import { usePersistentTableState } from './usePersistentTableState';
import { useCrudContext } from './CrudContext';
import { useState, useEffect, useCallback } from 'react';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';



interface Props<D extends FieldValues> {
  datasource: Datasource<D>
}


export function CrudList<D extends FieldValues>({ datasource }: Props<D>) {
  const { rootPath, workflow } = useCrudContext();
  const storageKey = `${workflow}_crud_edit_mode `
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  // État pour gérer la visibilité de la modale et les lignes sélectionnées
  const [open, setOpen] = useState(false);
  const [selectedRows, setSelectedRows] = useState<MRT_Row<D>[]>([]);
  const [isEditMode, setIsEditMode] = useState(() => {
    if (datasource.isReadOnly) return false;
    return sessionStorage.getItem(storageKey) === 'true';
  });
  // 2. Dans le composant, remplacer les 4 useState + 4 useEffect par :
  const {
    columnFilters, setColumnFilters,
    globalFilter, setGlobalFilter,
    sorting, setSorting,
    pagination, setPagination,
    showGlobalFilter, setShowGlobalFilter,
    showColumnFilters, setShowColumnFilters,
    columnVisibility, setColumnVisibility,
    density, setDensity,
    isFullScreen, setIsFullScreen,
  } = usePersistentTableState(datasource.queryKey);

  // READ : Récupération des données
  const { data, isLoading, isError } = useQuery({
    queryKey: datasource.queryKey,
    queryFn: datasource.fetchAll
  });

  // DELETE : Mutation pour supprimer
  const mutation = useMutation({
    mutationFn: datasource.delete,
    onSuccess: () => {
      // Invalide le cache et force un rafraîchissement automatique de la liste
      queryClient.invalidateQueries({ queryKey: datasource.queryKey });
    },
  });


  // Ferme la modale
  const handleClose = () => {
    setOpen(false);
    setSelectedRows([]);
  };

  // Exécute la suppression réelle
  const handleConfirmDelete = (table: MRT_TableInstance<D>) => {
    const ids = selectedRows.map(row => datasource.getId(row.original));

    // Un seul appel API, un seul onSuccess, un seul fetchAll
    mutation.mutate(ids);

    table.resetRowSelection();
    handleClose();
  };

  // Ouvre la modale et mémorise les lignes à supprimer
  const handleOpenModal = useCallback((table: MRT_TableInstance<D>) => {
    const rows = table.getSelectedRowModel().flatRows;
    if (rows.length > 0) {
      setSelectedRows(rows);
      setOpen(true);
    }
  }, []);

  const renderRowActions = useCallback(({ row, table }: { row: MRT_Row<D>, table: MRT_TableInstance<D> }) => {
    if (!datasource.isAction) return null

    const defaultActions = (
      <Box sx={{ display: 'flex', gap: '1rem' }}>
        {!(isEditMode && datasource.isAction) && (
          <Tooltip title="Voir">
            <span>
              <IconButton onClick={() => navigate(`${rootPath}/${datasource.getId(row.original)}`)}>
                <VisibilityIcon />
              </IconButton>
            </span>
          </Tooltip>
        )}

        {(isEditMode && datasource.isAction) && (
          <Tooltip title="Editer">
            <span>
              <IconButton onClick={() => navigate(`${rootPath}/${datasource.getId(row.original)}/edit`)}>
                <EditIcon />
              </IconButton>
            </span>
          </Tooltip>
        )}
      </Box >
    );

    // Si la datasource fournit une fonction personnalisée, on l'utilise en lui passant les actions par défaut
    if (datasource.renderRowActions) {
      return datasource.renderRowActions({ row, table, defaultActions, isEditMode });
    }

    return defaultActions;
  }, [datasource, isEditMode, navigate, rootPath]);

  const renderTopToolbarCustomActions = useCallback(({ table }: { table: MRT_TableInstance<D> }) => {
    if (!datasource.isTopToolbar) return null

    const defaultActions = (
      <Box sx={{ display: 'flex', gap: '1rem' }}>
        {datasource.isAction && !datasource.isReadOnly && (
          <>
            <Tooltip title={isEditMode ? "Passer en consultation" : "Passer en édition"}>
              <IconButton onClick={() => setIsEditMode(!isEditMode)}>
                {isEditMode ? <VisibilityIcon /> : <EditIcon />}
              </IconButton>
            </Tooltip>
            {isEditMode && (
              <>
                <Tooltip title="Add">
                  <span>
                    <IconButton onClick={() => navigate(`${rootPath}/new`)}>
                      <AddBoxIcon />
                    </IconButton>
                  </span>
                </Tooltip>
                <Tooltip title="Delete">
                  <span>
                    <IconButton
                      color="error"
                      onClick={() => handleOpenModal(table)}
                      disabled={table.getSelectedRowModel().flatRows.length === 0}>
                      <DeleteIcon />
                    </IconButton>
                  </span>
                </Tooltip>
              </>
            )}
          </>
        )}
      </Box>
    );

    if (datasource.renderTopToolbarCustomActions) {
      return datasource.renderTopToolbarCustomActions({ table, defaultActions, isEditMode });
    }
    return defaultActions;
  }, [isEditMode, navigate, datasource, handleOpenModal]);

  const table = useMaterialReactTable({
    initialState: {
      isLoading,
      density: 'compact', // 'compact' | 'comfortable' | 'spacious'
    },
    state: {
      isLoading,
      columnFilters,
      globalFilter,
      sorting,
      pagination,
      showGlobalFilter,
      showColumnFilters,
      columnVisibility,
      density,
      isFullScreen,
    },
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onShowGlobalFilterChange: setShowGlobalFilter,
    onShowColumnFiltersChange: setShowColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onDensityChange: setDensity,
    onIsFullScreenChange: setIsFullScreen,
    columns: datasource.columns,
    data: data ? data : [], //data must be memoized or stable (useState, useMemo, defined outside of this component, etc.)
    mrtTheme: (theme) => ({
      baseBackgroundColor: theme.palette.mode === 'dark' ?
        darken(theme.palette.background.default, 0.05) : theme.palette.background.default,
    }),
    renderTopToolbarCustomActions,
    enableRowActions: datasource.isAction,
    positionActionsColumn: 'last',
    renderRowActions,
    enableRowSelection: isEditMode,
    enableStickyHeader: true,
    enableStickyFooter: true,
    muiTablePaperProps: {
      sx: {
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        overflow: 'hidden',
      },
    },
    muiTableContainerProps: {
      sx: {
        flex: 1,
        overflow: 'auto',
      },
    },
  });

  useEffect(() => {
    if (!isEditMode) {
      table.resetRowSelection();
    }
  }, [isEditMode, table]);

  useEffect(() => {
    sessionStorage.setItem(storageKey, String(isEditMode));
  }, [isEditMode]);

  if (isError) return <div>Erreur lors du chargement</div>;

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', overflow: 'hidden' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2, gap: 1, flexShrink: 0 }}>
        {!datasource?.first && (
          <>
            <Tooltip title="Retour">
              <IconButton onClick={() => navigate(rootPath.split('/').slice(0, -2).join('/'))}>
                <ArrowBackIcon />
              </IconButton>
            </Tooltip>
            <Typography variant="h6" sx={{ flex: 1 }}>{datasource.title}</Typography>
          </>
        )}

      </Box>
      <Box sx={{ flex: 1, overflow: 'hidden', minHeight: 0 }}>
        <MaterialReactTable table={table} />
      </Box>
      {/* Modale de confirmation MUI */}
      <Dialog
        open={open}
        onClose={handleClose}
        aria-labelledby="alert-dialog-title"
        aria-describedby="alert-dialog-description"
      >
        <DialogTitle id="alert-dialog-title">
          {"Confirmer la suppression"}
        </DialogTitle>
        <DialogContent>
          <DialogContentText id="alert-dialog-description">
            Êtes-vous sûr de vouloir supprimer {selectedRows.length} élément(s) ?
            Cette action est irréversible.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleClose}>Annuler</Button>
          <Button
            onClick={() => handleConfirmDelete(table)}
            color="error"
            autoFocus
          >
            Supprimer
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )

}