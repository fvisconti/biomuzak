// frontend/src/App.test.js
import { render, screen, waitFor } from '@testing-library/react';
import Library from './components/Library';
import * as api from './api';

// Mock the API used by Library so the component renders deterministically in tests
jest.spyOn(api, 'getLibrary').mockResolvedValue({ data: [] });

test('renders My Library header', async () => {
  render(<Library />);
  await waitFor(() => {
    expect(screen.getByText(/my library/i)).toBeInTheDocument();
  });
});
