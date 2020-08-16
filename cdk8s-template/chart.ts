import { Construct } from 'constructs';
import { Chart } from 'cdk8s';
import { Deployment } from './imports/k8s';

export default class MyChart extends Chart {
  constructor(scope: Construct, name: string) {
    super(scope, name);

    const labels = {
        "app": "foobarhoge"
    };

    new Deployment(this, "app", {
        spec: {
            replicas: 1,
            selector: {
                matchLabels: labels,
            },
            template: {
                metadata: { labels },
                spec: {
                    containers: [
                        {
                            name: "main",
                            image: "paulbouwer/hello-kubernetes:1.7",
                            ports: [
                                { containerPort: 8080 },
                            ],
                        }
                    ]
                }
            }
        }
    });

    // define resources here

  }
}
