import { Container, Flex, Heading, SimpleGrid, Text } from "@chakra-ui/react";
import SystemOverview from "./components/SystemOverview";
import FormLibrary from "./components/FormLibrary";
import WorkflowDesigner from "./components/WorkflowDesigner";
import TicketDashboard from "./components/TicketDashboard";

function App() {
  return (
    <Container maxW="6xl" py={8} gap={6}>
      <Flex direction="column" gap={6}>
        <Heading>PFlow 控制台</Heading>
        <Text color="gray.600">
          拖拽表单、流程编排、工单生命周期一体化的流程引擎 &amp; 工单管理平台。
        </Text>
        <SystemOverview />
        <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6}>
          <FormLibrary />
          <WorkflowDesigner />
        </SimpleGrid>
        <TicketDashboard />
      </Flex>
    </Container>
  );
}

export default App;
