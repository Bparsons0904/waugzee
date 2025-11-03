import {
  ArcElement,
  CategoryScale,
  Chart,
  Colors,
  Legend,
  LinearScale,
  LineController,
  LineElement,
  PieController,
  PointElement,
  TimeScale,
  Title,
  Tooltip,
} from "chart.js";

export function registerChartComponents() {
  Chart.register(
    CategoryScale,
    LinearScale,
    TimeScale,
    LineController,
    LineElement,
    PointElement,
    PieController,
    ArcElement,
    Title,
    Tooltip,
    Legend,
    Colors,
  );
}
