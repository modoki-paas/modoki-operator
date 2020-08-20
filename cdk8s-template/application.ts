

export interface ApplicationSpec {
    domains: string[];
    image: string;
    command?: string[];
    args?: string[];
    attributes?: {[key: string]: string};
};

export interface ApplicationStatus {
    domains: string[];
    status: "deployed" | "progressing" | "failed" | "error";
    message: string;
    resources: ApplicationResource[];
};

export interface ObjectMeta {
    name: string;
    namespace: string;
    labels: {[key: string]: string};
    annotations: {[key: string]: string};
}

export interface ApplicationResource {
    kind: string;
    apiVersion: string;
    name: string;
    namespace: string;
};

export interface Application {
    kind: string;
    apiVersion: string;
    metadata: ObjectMeta;
    spec: ApplicationSpec;
    status: ApplicationStatus
};