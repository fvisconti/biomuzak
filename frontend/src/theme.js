import { extendTheme } from '@chakra-ui/react';

const theme = extendTheme({
  config: {
    initialColorMode: 'dark',
    useSystemColorMode: false,
  },
  fonts: {
    heading: `'Iosevka', 'Consolas', 'Monaco', 'Courier New', monospace`,
    body: `'Iosevka', 'Consolas', 'Monaco', 'Courier New', monospace`,
  },
  styles: {
    global: (props) => ({
      body: {
        bg: props.colorMode === 'dark' ? '#0d0d0d' : 'gray.50',
        color: props.colorMode === 'dark' ? 'white' : 'gray.900',
      },
    }),
  },
  colors: {
    brand: {
      50: '#e3f2fd',
      100: '#bbdefb',
      400: '#42a5f5', // Added for icon color
      500: '#2196f3', // Added for button default
      600: '#1e88e5', // Added for button hover
      900: '#0d47a1',
    },
    gray: {
      800: '#1a1a1a', // Panel background
      900: '#0d0d0d', // Main background
    },
  },
  components: {
    Button: {
      defaultProps: {
        colorScheme: 'gray',
        variant: 'outline',
      },
      baseStyle: {
        borderRadius: 0, // Sharp edges for source code look
        _focus: {
          boxShadow: 'none',
        },
      },
    },
  },
});

export default theme;
