import { useState, useCallback, useEffect } from 'react';
import { apiRequest } from '../api/client.js';

const EMPTY_PORTFOLIO = { items: [], totals: {}, has_price_data: false };

/**
 * Loads the portfolio, price history, and signal list together, since
 * every view needs some combination of the three and they are cheap
 * enough to refetch as a unit.
 */
export function useGoldData() {
  const [portfolio, setPortfolio] = useState(EMPTY_PORTFOLIO);
  const [prices, setPrices] = useState([]);
  const [signals, setSignals] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const refreshData = useCallback(async () => {
    try {
      const [portfolioData, pricesData, signalsData] = await Promise.all([
        apiRequest('/api/portfolio'),
        apiRequest('/api/prices'),
        apiRequest('/api/signals'),
      ]);
      setPortfolio(portfolioData || EMPTY_PORTFOLIO);
      setPrices(pricesData || []);
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
