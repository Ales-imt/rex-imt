import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router";
import { CrudList } from "./List";
import { Form } from "./Form";
import type { CrudMode, Datasource } from "./def";
import type { DefaultValues, FieldValues } from "react-hook-form";
import { CrudProvider } from "./CrudContext";


interface Props<D extends FieldValues> {
    mode: CrudMode
    datasource: Datasource<D>
    workflow: string
    rootPath: string
}

export function Crud<D extends FieldValues>({ datasource, mode, workflow, rootPath }: Props<D>) {

    const { id } = useParams(); // Récupère l'ID depuis l'URL

    const { data, isLoading, error } = useQuery<D>({
        queryKey: [...datasource.queryKey, id],
        queryFn: () => datasource.fetch(id!),
        enabled: !!id && (mode === 'show' || mode === 'edit'),
    });

    // On wrappe le contenu dans le Provider pour rendre rootPath et workflow accessibles
    const content = (() => {
    if (mode === 'list') {
        return <CrudList datasource={datasource} />;
    } else if (mode === 'create') {
        return <Form datasource={datasource} initialData={datasource.emptyValue} mode="create" />;
    }

    if (isLoading) return <p>Chargement des données...</p>;
    if (error) return <p>Erreur lors de la récupération.</p>;

    if (mode === 'show') {
        if (!data) return <p>Données introuvables.</p>;
        return <Form datasource={datasource} initialData={data as DefaultValues<D>} mode="show" />;
    } else if (mode === 'edit') {
        if (!data) return <p>Données introuvables.</p>;
        return <Form datasource={datasource} initialData={data as DefaultValues<D>} mode="edit" />;
    } else {
        return <p>Page non trouvée</p>;
    }
    })();

    return (
        <CrudProvider value={{ rootPath, workflow }}>
            {content}
        </CrudProvider>
    );

}