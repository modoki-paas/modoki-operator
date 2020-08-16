import { App } from 'cdk8s';
import MyChart from "./chart";

const app = new App();
const array = new MyChart(app, 'cdk8s-template').toJson();
console.log(JSON.stringify(array));
