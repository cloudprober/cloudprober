import type {
  Reporter, FullConfig, Suite, TestCase, TestStep, TestResult
} from '@playwright/test/reporter';

const info = (string: string) => process.stderr.write("INFO "+string+'\n');
const warning = (string: string) => process.stderr.write("WARNING "+string+'\n');

const print = (string: string) => process.stdout.write(string+'\n');

class CloudproberReporter implements Reporter {
  onBegin(config: FullConfig, suite: Suite) {
    info(`Starting the run with ${suite.allTests().length} tests`);
  }

  onTestBegin(test: TestCase) {
    info(`Starting test ${test.title}`);
  }

  onStepEnd(test: TestCase, result: TestResult, step: TestStep) {
    var testTitle = test.title.replace(/ /g,"_");
    var stepTitle = step.title.replace(/ /g,"_");
    if (step.category === 'test.step') {
      if (step.error !== null) {
        warning(`Test step ${stepTitle} of test ${testTitle} failed with error: ${JSON.stringify(step.error)}`);
      }
      print(`test_step_status{test="${testTitle}",step="${stepTitle}",status="${result.status}"} 1`);
      print(`test_step_latency{test="${testTitle}",step="${stepTitle}",status="${result.status}"} ${step.duration*1000}`);
    }
  }

  onTestEnd(test: TestCase, result: TestResult) {
    if (result.status !== "passed") {
      warning(`Test ${test.title} failed with errors: ${JSON.stringify(result.errors)}`);
    }
    var title = test.title.replace(/ /g,"_");
    print(`test_status{test="${title}",status="${result.status}"} 1`);
    print(`test_latency{test="${title}",status="${result.status}"} ${result.duration*1000}`);
  }
}
export default CloudproberReporter;