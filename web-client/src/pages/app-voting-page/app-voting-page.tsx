import React from 'react';
import {createStyles, makeStyles, TextField, Theme} from '@material-ui/core';
import AppHeadline from '../../components/app-headline/app-headline';
import {forms} from '../../theme/app-theme-constants';
import AppButton from '../../components/app-button/app-button';
import * as yup from 'yup';
import {useFormik} from 'formik';
import AppVotingCard from '../../components/app-voting-card/app-voting-card';

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    enterVote: {
      display: 'grid',
      gridTemplateColumns: '1fr auto',
      gridColumnGap: theme.spacing(1),
      marginBottom: theme.spacing(forms.lastField)
    }
  })
);

const hashLen = 32;
const validationSchema = yup.object({
  voteHash: yup
    .string()
    .min(hashLen, `Длина идентификатора ровно ${hashLen} символа`)
    .max(hashLen, `Длина идентификатора ровно ${hashLen} символа`)
    .required(),
});

function AppVotingPage() {
  const classes = useStyles();
  const formik = useFormik({
    initialValues: {
      voteHash: '',
    },
    validationSchema: validationSchema,
    onSubmit: (values) => {
      alert(JSON.stringify(values));
    }
  });
  return (
    <>
      <AppHeadline>Голосования</AppHeadline>
      <form
        className={classes.enterVote}
        onSubmit={formik.handleSubmit}
      >
        <TextField
          required
          id="voteHash"
          label="Ввести идентификатор голосования"
          placeholder="hex строка длиной 32 символа"
          type="text"
          fullWidth
          value={formik.values.voteHash}
          onChange={formik.handleChange}
          error={formik.touched.voteHash && !!formik.errors.voteHash}
          helperText={formik.touched.voteHash && formik.errors.voteHash}
        >
        </TextField>
        <AppButton
          type="submit"
        >
          Открыть
        </AppButton>
      </form>
      <AppVotingCard hash="03b805fab5e8ec2eee92496925b8067a" myVotes={1337}/>
      <AppVotingCard hash="03b805fab5e8ec2eee92496925b8067a" myVotes={10}/>
      <AppVotingCard hash="03b805fab5e8ec2eee92496925b8067a" myVotes={2}/>
      <AppVotingCard hash="03b805fab5e8ec2eee92496925b8067a" myVotes={1000}/>
    </>
  )
}

export default AppVotingPage
