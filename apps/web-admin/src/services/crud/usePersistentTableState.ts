import { useState, useEffect } from 'react';
import type { QueryKey } from '@tanstack/react-query';
import type {
  MRT_ColumnFiltersState,
  MRT_SortingState,
  MRT_PaginationState,
  MRT_VisibilityState,
  MRT_DensityState,
} from 'material-react-table';

export function usePersistentTableState(queryKey: QueryKey) {
  const key = JSON.stringify(queryKey);
  const filtersKey         = `${key}_col_filters`;
  const globalFilterKey    = `${key}_global_filter`;
  const sortingKey         = `${key}_sorting`;
  const paginationKey      = `${key}_pagination`;
  const showSearchKey      = `${key}_show_search`;
  const showFiltersKey     = `${key}_show_filters`;
  const columnVisKey       = `${key}_col_visibility`;
  const densityKey         = `${key}_density`;

  const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>(() => {
    const saved = sessionStorage.getItem(filtersKey);
    return saved ? JSON.parse(saved) : [];
  });

  const [globalFilter, setGlobalFilter] = useState<string>(() => {
    return sessionStorage.getItem(globalFilterKey) ?? '';
  });

  const [sorting, setSorting] = useState<MRT_SortingState>(() => {
    const saved = sessionStorage.getItem(sortingKey);
    return saved ? JSON.parse(saved) : [];
  });

  const [pagination, setPagination] = useState<MRT_PaginationState>(() => {
    const saved = sessionStorage.getItem(paginationKey);
    return saved ? JSON.parse(saved) : { pageIndex: 0, pageSize: 10 };
  });

  const [showGlobalFilter, setShowGlobalFilter] = useState<boolean>(() => {
    return sessionStorage.getItem(showSearchKey) === 'true';
  });

  const [showColumnFilters, setShowColumnFilters] = useState<boolean>(() => {
    return sessionStorage.getItem(showFiltersKey) === 'true';
  });

  const [columnVisibility, setColumnVisibility] = useState<MRT_VisibilityState>(() => {
    const saved = sessionStorage.getItem(columnVisKey);
    return saved ? JSON.parse(saved) : {};
  });

  const [density, setDensity] = useState<MRT_DensityState>(() => {
    return (sessionStorage.getItem(densityKey) as MRT_DensityState) ?? 'compact';
  });

  const [isFullScreen, setIsFullScreen] = useState<boolean>(() => {
    // On ne restaure pas le plein écran : une page rechargée en fullscreen est déroutant
    return false;
  });

  useEffect(() => { sessionStorage.setItem(filtersKey,      JSON.stringify(columnFilters));   }, [columnFilters]);
  useEffect(() => { sessionStorage.setItem(globalFilterKey, globalFilter ?? '');              }, [globalFilter]);
  useEffect(() => { sessionStorage.setItem(sortingKey,      JSON.stringify(sorting));         }, [sorting]);
  useEffect(() => { sessionStorage.setItem(paginationKey,   JSON.stringify(pagination));      }, [pagination]);
  useEffect(() => { sessionStorage.setItem(showSearchKey,   String(showGlobalFilter));        }, [showGlobalFilter]);
  useEffect(() => { sessionStorage.setItem(showFiltersKey,  String(showColumnFilters));       }, [showColumnFilters]);
  useEffect(() => { sessionStorage.setItem(columnVisKey,    JSON.stringify(columnVisibility));}, [columnVisibility]);
  useEffect(() => { sessionStorage.setItem(densityKey,      density);                         }, [density]);
  // isFullScreen volontairement non persisté

  return {
    columnFilters,      setColumnFilters,
    globalFilter,       setGlobalFilter,
    sorting,            setSorting,
    pagination,         setPagination,
    showGlobalFilter,   setShowGlobalFilter,
    showColumnFilters,  setShowColumnFilters,
    columnVisibility,   setColumnVisibility,
    density,            setDensity,
    isFullScreen,       setIsFullScreen,
  };
}