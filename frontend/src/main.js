import { Events } from "@wailsio/runtime";
import Alpine from "alpinejs";
import humanizeDuration from "humanize-duration";
import "./style.css";

const shortEnglishHumanizer = humanizeDuration.humanizer({
  language: "shortEn",
  languages: {
    shortEn: {
      y: () => "y",
      mo: () => "mo",
      w: () => "w",
      d: () => "d",
      h: () => "h",
      m: () => "m",
      s: () => "s",
      ms: () => "ms",
    },
  },
});

window.Alpine = Alpine;

Alpine.store("stats", {
  ExpText: "-",
  PercentText: "-",
  TimeToLevelUpText: "-",
});

Alpine.start();

Events.On("updateXP", (event) => {
  console.log("-------", { event });
  const data = event.data[0];
  const store = Alpine.store("stats");
  store.ExpText = ((data.ExpPerSecond ?? 0.0) * 60).toFixed(2) + "/min";
  store.PercentText = ((data.PercentPerSecond ?? 0.0) * 60).toFixed(2) + "/min";

  const timeToLevelUpInMs = Math.round(
    (data.TimeToLevelUpPerSecond ?? 0.0) * 1_000,
  );
  store.TimeToLevelUpText = shortEnglishHumanizer(timeToLevelUpInMs, {
    largest: 2,
    maxDecimalPoints: 2,
    spacer: "",
  });
});
