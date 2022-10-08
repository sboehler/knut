import React from "react";
import { Counter } from "./features/counter/Counter";

function App() {
  return  <React.Suspense fallback={<div>Loading...</div>}>
    <Counter />;
  </React.Suspense> 
}
export default App;
