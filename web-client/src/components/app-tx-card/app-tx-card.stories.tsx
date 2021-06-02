import AppTxCard, {AppTxCardProps} from './app-tx-card';
import {Story} from '@storybook/react';

export default {
  title: 'AppTxCard',
  component: AppTxCard,
}

const Template: Story<AppTxCardProps> = (args) => (<AppTxCard {...args}/>);

export const In = Template.bind({});
In.args = {
  type: 'in',
  name: '03b805fab5e8ec2eee92496925b8067a',
  value: 1337,
};

export const Out = Template.bind({});
Out.args = {
  ...In.args,
  type: 'out',
};

export const VoteIn = Template.bind({});
VoteIn.args = {
  ...In.args,
  type: 'voteIn',
};

export const VoteOut = Template.bind({});
VoteOut.args = {
  ...In.args,
  type: 'voteOut',
};
