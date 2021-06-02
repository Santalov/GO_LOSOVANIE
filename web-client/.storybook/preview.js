import React from 'react';
import '../src/styles/App.scss';
import {addDecorator} from '@storybook/react';
import AppThemeProvider from '../src/theme/app-theme-provider';
import makeStyles from '@material-ui/core/styles/makeStyles';
import withTheme from '@material-ui/core/styles/withTheme';

function WrapRaw({ theme, children }) {
  console.log(theme);
  const classes = makeStyles({
    root: {
      backgroundColor: theme.palette.background.default,
      color: theme.palette.text.primary,
      position: "absolute",
      top: 0,
      left: 0,
      bottom: 0,
      right: 0,
      padding: 30,
    },
    notroot: {
      width: 600,
    },
  })();
  return (
    <div className={classes.root}>
    <div className={classes.notroot}>{children}</div>
    </div>
);
}

const Wrap = withTheme(WrapRaw);

addDecorator((storyFn) => <Wrap>{storyFn()}</Wrap>);
addDecorator((storyFn) => <AppThemeProvider>{storyFn()}</AppThemeProvider>);
