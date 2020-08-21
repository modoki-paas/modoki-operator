import { Construct } from 'constructs';
import { Chart } from 'cdk8s';
import * as k8s from '../imports/k8s';
import { Application } from '../application';

export interface Config {
    ingressNamespace: string;
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

    const defaultConfig: Config = {
        ingressNamespace: "modoki-operator-system",
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

    config.ingressNamespace = config.ingressNamespace ?? "modoki-operator-system";
    config.tlsSecretName = config.tlsSecretName ?? "ingress-secret";
    config.serviceDomain = config.serviceDomain ?? "svc.cluster.local";

    const labels = {
        "app": `modoki-${app.metadata.name}-app`,
    };
    const annotations = {
        "modoki.tsuzu.dev": `modoki-${app.metadata.name}-app`,
    };
    const metadata = {
        labels,
        annotations,
        namespace: app.metadata.namespace,
    }

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
                    automountServiceAccountToken: false,
                    containers: [
                        {
                            name: "main",
                            image:  app.spec.image,
                            command: app.spec.command,
                            args: app.spec.args,
                            ports: [
                                { containerPort: 80 },
                            ],
                            env: [
                                {name: "PORT", value: "80"},
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
                    port: 80,
                    targetPort: 80,
                    protocol: "TCP",
                }
            ],
            selector: labels,
            type: "ClusterIP",
        },
    });

    const sanitizeName = (s: string) => s.replace(/-/g, "--");

    const externalSVC = new k8s.Service(this, "external-svc", {
        metadata: {
            ...metadata,
            namespace: config.ingressNamespace,
            name: "modoki-extsvc-" + sanitizeName(app.metadata.namespace) + "-" + sanitizeName(app.metadata.name),
        },
        spec: {
            externalName: `${svc.name}.${app.metadata.namespace}.${config.serviceDomain}`,
            type: "ExternalName",
        },
    });

    new k8s.Ingress(this, "ingress", {
        metadata: {
            ...metadata,
            namespace: config.ingressNamespace,
            name: "modoki-ingress-" + sanitizeName(app.metadata.namespace) + "-" + sanitizeName(app.metadata.name),
            annotations: {
                ...annotations,
                ...config.ingressAnnotations,
            }
        },
        spec: {
            backend: {
                serviceName: externalSVC.name,
                servicePort: 80,
            },
            tls: [
                {
                    hosts: app.spec.domains,
                    secretName: config.tlsSecretName,
                }
            ]
        }
    })

    // define resources here

  }
}
