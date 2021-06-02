import AppHeadline from '../../components/app-headline/app-headline';
import {createStyles, makeStyles, Theme} from '@material-ui/core';
import React from 'react';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({})
);

function AppHistoryPage() {
  return (
    <>
      <AppHeadline>История</AppHeadline>
    </>
  )
}
