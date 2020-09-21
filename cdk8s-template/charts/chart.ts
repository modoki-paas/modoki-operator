import { Construct } from 'constructs';
import { Chart } from 'cdk8s';
import * as k8s from '../imports/k8s';
import { Application } from '../application';

export interface Config {
    ingressNamespace: string;
    oauth2ServiceName: string;
    oauth2ServiceNamespace: string;
    tlsSecretName: string;
    serviceDomain: string;
    ingressAnnotations: { [key: string]: string };
};

interface MyChartProp {
    app: Application;
    config: Partial<Config>;
}

export default class MyChart extends Chart {
  constructor(scope: Construct, name: string, prop: MyChartProp) {
    const {app, config: rawConfig} = prop;
    super(scope, name);

    if(!app.spec.image?.length) {
        app.spec.image = "modokipaas/no-app"
    }

    const defaultConfig: Config = {
        ingressNamespace: "modoki-operator-system",
        oauth2ServiceName: "oauth2-proxy",
        oauth2ServiceNamespace: "modoki-operator-system",
        tlsSecretName: "ingress-secret",
        serviceDomain: "svc.cluster.local",
        ingressAnnotations: {
            "kubernetes.io/ingress.class": "nginx",
        }
    }

    const config: Config = {
        ...defaultConfig,
        ...rawConfig
    };

    const oauth2Injection = (app.spec.attributes?.["modoki.tsuzu.dev/oauth2"] ?? "") === "default";

    if(oauth2Injection) {
        config.ingressAnnotations["nginx.ingress.kubernetes.io/auth-url"] = "https://$host/oauth2/auth"
        config.ingressAnnotations["nginx.ingress.kubernetes.io/auth-signin"] = "https://$host/oauth2/start?rd=$escaped_request_uri"
    }

    const labels = {
        "modoki-app": `${app.metadata.name}`,
    };
    const annotations = {
        "modoki.tsuzu.dev": `modoki-${app.metadata.name}-app`,
    };
    const metadata = {
        labels,
        annotations,
        namespace: app.metadata.namespace,
    }

    const port = 8080;

    new k8s.Deployment(this, "main-deployment", {
        metadata,
        spec: {
            replicas: 1,
            selector: {
                matchLabels: labels,
            },
            template: {
                metadata: { labels },
                spec: {
                    serviceAccount: app.spec.serviceAccount?.length ? app.spec.serviceAccount : undefined,
                    serviceAccountName: app.spec.serviceAccount?.length ? app.spec.serviceAccount : undefined,
                    imagePullSecrets: app.spec.imagePullSecret?.length ? [{name: app.spec.imagePullSecret}] : undefined,
                    automountServiceAccountToken: false,
                    containers: [
                        {
                            name: "main",
                            image:  app.spec.image,
                            command: app.spec.command,
                            args: app.spec.args,
                            ports: [
                                { containerPort: port },
                            ],
                            env: [
                                {name: "PORT", value: `${port}`},
                            ]
                        }
                    ]
                }
            }
        }
    });

    const svc = new k8s.Service(this, "service", {
        metadata,
        spec: {
            ports: [
                {
                    name: "http",
                    port: port,
                    targetPort: port,
                    protocol: "TCP",
                }
            ],
            selector: labels,
            type: "ClusterIP",
        },
    });

    new k8s.Ingress(this, "ingress", {
        metadata: {
            ...metadata,
            annotations: {
                ...annotations,
                ...config.ingressAnnotations,
            }
        },
        spec: {
            rules: app.spec.domains.map(x => ({
                host: x,
                http: {
                    paths: [{
                        backend: {
                            serviceName: svc.name,
                            servicePort: port,
                        },
                        path: "/",
                    }],
                }
            })),
            tls: [{
                hosts: app.spec.domains,
            }]
        }
    });

    if (oauth2Injection) {
        const svc = new k8s.Service(this, "oauth2-service", {
            metadata,
            spec: {
                type: "ExternalName",
                externalName: `${config.oauth2ServiceName}.${config.oauth2ServiceNamespace}.${config.serviceDomain}`,
            },
        });

        new k8s.Ingress(this, "ingress", {
            metadata: {
                ...metadata,
                annotations: {
                    ...annotations,
                    ...config.ingressAnnotations,
                }
            },
            spec: {
                rules: app.spec.domains.map(x => ({
                    host: x,
                    http: {
                        paths: [{
                            backend: {
                                serviceName: svc.name,
                                servicePort: 4180,
                            },
                            path: "/oauth2",
                        }],
                    }
                })),
                tls: [{
                    hosts: app.spec.domains,
                }]
            }
        })
    }

    // define resources here

  }
}
