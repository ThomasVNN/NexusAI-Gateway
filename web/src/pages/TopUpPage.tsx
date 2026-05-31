import { useState } from 'react';
import { CreditCard, Zap, Plus } from 'lucide-react';

interface TopUpOption {
  id: string;
  amount: number;
  bonus?: number;
  popular?: boolean;
}

const topUpOptions: TopUpOption[] = [
  { id: '1', amount: 50 },
  { id: '2', amount: 100 },
  { id: '3', amount: 250, bonus: 10 },
  { id: '4', amount: 500, bonus: 15, popular: true },
  { id: '5', amount: 1000, bonus: 25 },
];

export function TopUpPage() {
  const [selectedAmount, setSelectedAmount] = useState<number>(100);
  const [customAmount, setCustomAmount] = useState<string>('');
  const [processing, setProcessing] = useState(false);

  const currentBalance = 847.52;

  const handleTopUp = async () => {
    setProcessing(true);
    await new Promise((r) => setTimeout(r, 2000));
    alert(`Top-up of $${selectedAmount} initiated! This would redirect to payment gateway in production.`);
    setProcessing(false);
  };

  const effectiveAmount = () => {
    const option = topUpOptions.find((o) => o.amount === selectedAmount);
    if (option?.bonus) {
      return selectedAmount * (1 + option.bonus / 100);
    }
    return selectedAmount;
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Page Header */}
      <div className="text-center">
        <h1 className="text-2xl font-bold text-text-primary">Top-Up Your Balance</h1>
        <p className="text-sm text-text-tertiary mt-1">Add credits to your account to continue using the Gateway</p>
      </div>

      {/* Current Balance */}
      <div className="bg-gradient-to-br from-accent-primary/20 to-accent-secondary/10 border border-accent-primary/30 rounded-xl p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-text-tertiary">Current Balance</p>
            <p className="text-4xl font-bold text-text-primary mt-1">${currentBalance.toFixed(2)}</p>
          </div>
          <div className="w-16 h-16 rounded-2xl bg-accent-primary/20 flex items-center justify-center">
            <CreditCard className="w-8 h-8 text-accent-primary" />
          </div>
        </div>
      </div>

      {/* Top-Up Options */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6">
        <h2 className="text-lg font-semibold text-text-primary mb-4">Select Amount</h2>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
          {topUpOptions.map((option) => (
            <button
              key={option.id}
              onClick={() => setSelectedAmount(option.amount)}
              className={`relative p-4 rounded-xl border-2 transition-all ${
                selectedAmount === option.amount
                  ? 'border-accent-primary bg-accent-primary/10'
                  : 'border-border-subtle hover:border-border-default'
              }`}
            >
              {option.popular && (
                <span className="absolute -top-2 left-1/2 -translate-x-1/2 px-2 py-0.5 bg-accent-primary text-white text-xs font-medium rounded-full">
                  Popular
                </span>
              )}
              <p className="text-2xl font-bold text-text-primary">${option.amount}</p>
              {option.bonus && (
                <p className="text-sm text-success mt-1">+{option.bonus}% bonus</p>
              )}
            </button>
          ))}
        </div>

        {/* Custom Amount */}
        <div className="mt-4">
          <label className="block text-sm font-medium text-text-secondary mb-1">Or enter custom amount</label>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted">$</span>
              <input
                type="number"
                value={customAmount}
                onChange={(e) => {
                  setCustomAmount(e.target.value);
                  if (e.target.value) {
                    setSelectedAmount(Number(e.target.value));
                  }
                }}
                placeholder="0.00"
                className="w-full pl-8 pr-4 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Summary */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6">
        <h2 className="text-lg font-semibold text-text-primary mb-4">Summary</h2>
        <div className="space-y-3">
          <div className="flex justify-between text-sm">
            <span className="text-text-secondary">Top-up amount</span>
            <span className="text-text-primary">${selectedAmount.toFixed(2)}</span>
          </div>
          {topUpOptions.find((o) => o.amount === selectedAmount)?.bonus && (
            <div className="flex justify-between text-sm">
              <span className="text-text-secondary">Bonus</span>
              <span className="text-success">
                +${(selectedAmount * (topUpOptions.find((o) => o.amount === selectedAmount)?.bonus || 0) / 100).toFixed(2)}
              </span>
            </div>
          )}
          <div className="border-t border-border-subtle pt-3 flex justify-between">
            <span className="font-medium text-text-primary">You will receive</span>
            <span className="text-xl font-bold text-accent-primary">${effectiveAmount().toFixed(2)}</span>
          </div>
        </div>
      </div>

      {/* Payment Button */}
      <button
        onClick={handleTopUp}
        disabled={processing}
        className="w-full flex items-center justify-center gap-2 px-6 py-4 bg-accent-primary text-white rounded-xl hover:bg-accent-hover transition-colors disabled:opacity-50"
      >
        {processing ? (
          <>
            <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            Processing...
          </>
        ) : (
          <>
            <Zap className="w-5 h-5" />
            Top-Up ${selectedAmount.toFixed(2)}
          </>
        )}
      </button>

      {/* Payment Methods */}
      <div className="text-center">
        <p className="text-xs text-text-muted mb-2">Secure payment powered by Stripe</p>
        <div className="flex items-center justify-center gap-4">
          {['Visa', 'Mastercard', 'Amex', 'PayPal'].map((method) => (
            <div
              key={method}
              className="px-3 py-1.5 bg-bg-secondary border border-border-subtle rounded text-xs text-text-tertiary"
            >
              {method}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
