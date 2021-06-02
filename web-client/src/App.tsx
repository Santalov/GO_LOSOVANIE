import React from 'react';
import './styles/App.scss';
import AppThemeProvider from './theme/app-theme-provider';
import AppBackground from './components/app-background/app-background';
import {BrowserRouter as Router, Route, Switch} from 'react-router-dom';
import AppDashboardPage from './pages/app-dashboard-page/app-dashboard-page';

function App() {
  return (
    <AppThemeProvider>
      <AppBackground>
        <Router basename={process.env.PUBLIC_URL}>
          <Switch>
            <Route path="/">
              <AppDashboardPage/>
            </Route>
          </Switch>
        </Router>
      </AppBackground>
    </AppThemeProvider>
  );
}

export default App;
