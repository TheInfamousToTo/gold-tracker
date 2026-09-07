import { useState, useCallback, useEffect } from 'react';
import { apiRequest } from '../api/client.js';
import { withRates } from '../lib/prices.js';

const EMPTY_PORTFOLIO = { items: [], totals: {}, has_price_data: false };

/**
 * Loads the portfolio, both price series, and the signal list together,
 * since every view needs some combination of them and they are cheap
 * enough to refetch as a unit.
 *
 * Gold and silver are separate tables behind one endpoint, so they are
 * two requests. Both are normalised to a common `rate` field here, so
 * no component downstream has to know which column its metal uses.
 */
export function useGoldData() {
  const [portfolio, setPortfolio] = useState(EMPTY_PORTFOLIO);
  const [prices, setPrices] = useState({ gold: [], silver: [] });
  const [signals, setSignals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const refreshData = useCallback(async () => {
    try {
      const [portfolioData, goldPrices, silverPrices, signalsData] = await Promise.all([
        apiRequest('/api/portfolio'),
        apiRequest('/api/prices'),
        apiRequest('/api/prices?metal=silver'),
        apiRequest('/api/signals'),
      ]);
      setPortfolio(portfolioData || EMPTY_PORTFOLIO);
      setPrices({
        gold: withRates(goldPrices, 'gold'),
        silver: withRates(silverPrices, 'silver'),
      });
      setSignals(signalsData || []);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refreshData();
  }, [refreshData]);

  return { portfolio, prices, signals, loading, error, refreshData };
}
