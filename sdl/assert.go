package sdl

import "runtime"
import "fmt"
import "os"
import "sync"
import "log/slog"

// SDL_ASSERT_LEVEL can be set at compile time using -X sdl.SDL_ASSERT_LEVEL=1, etc
var SDL_ASSERT_LEVEL = 2

func SDL_TriggerBreakpoint() {
	runtime.Breakpoint()
}

func SDL_FILE() string {
	_, fileName, _, _ := runtime.Caller(1)
	return fileName
}

func SDL_LINE() int {
	_, _, line, _ := runtime.Caller(1)
	return line
}

func SDL_disabled_assert(condition bool) {

}

type SDL_AssertState int

const (
	SDL_ASSERTION_RETRY         SDL_AssertState = iota /**< Retry the assert immediately. */
	SDL_ASSERTION_BREAK                                /**< Make the debugger trigger a breakpoint. */
	SDL_ASSERTION_ABORT                                /**< Terminate the program. */
	SDL_ASSERTION_IGNORE                               /**< Ignore the assert. */
	SDL_ASSERTION_ALWAYS_IGNORE                        /**< Ignore the assert from now on. */
)

type SDL_AssertData struct {
	AlwaysIgnore bool
	TriggerCount int
	Condition    string
	Filename     string
	Linenum      int
	Function     string
	Next         *SDL_AssertData
}

/*
 * Never call this directly. Use the SDL_assert function instead.
 *
 * Parameters:
 * - data assert data structure
 * - func function name
 * - file file name
 * - line line number
 *
 * Returns assert state
 *
 * This function is available since SDL 3.0.0.
 */
func SDL_ReportAssertion(data *SDL_AssertData, fn string, file string, line int) SDL_AssertState {
	var state SDL_AssertState = SDL_ASSERTION_IGNORE
	var assertionRunning = 0
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	if data.TriggerCount == 0 {
		data.Function = fn
		data.Filename = file
		data.Linenum = line
	}

	SDL_AddAssertionToReport(data)

	assertionRunning++
	if assertionRunning > 1 { /* assert during assert! Abort. */
		if assertionRunning == 2 {
			SDL_AbortAssertion()
		} else if assertionRunning == 3 { /* Abort asserted! */
			SDL_ExitProcess(42)
		} else {
			runtime.Gosched()
		}
	}

	if !data.AlwaysIgnore {
		state = assertionHandler(data, assertionData)
	}

	switch state {
	case SDL_ASSERTION_ALWAYS_IGNORE:
		state = SDL_ASSERTION_IGNORE
		data.AlwaysIgnore = true
		break

	case SDL_ASSERTION_IGNORE:
	case SDL_ASSERTION_RETRY:
	case SDL_ASSERTION_BREAK:
		break /* macro handles these. */

	case SDL_ASSERTION_ABORT:
		SDL_AbortAssertion()
		/*break;  ...shouldn't return, but oh well. */
	}

	assertionRunning--

	return state
}

func SDL_AssertBreakpoint() {
	SDL_TriggerBreakpoint()
}

func SDL_enabled_assert(condition bool) {
	for !condition {
		var sdl_assert_data = SDL_AssertData{}
		pc, file, line, _ := runtime.Caller(2)
		fn := runtime.FuncForPC(pc).Name()
		state := SDL_ReportAssertion(&sdl_assert_data, fn, file, line)
		if state == SDL_ASSERTION_RETRY {
			continue /* go again. */
		} else if state == SDL_ASSERTION_BREAK {
			SDL_AssertBreakpoint()
		}
		break /* not retrying. */
	}
}

func SDL_assert(condition bool) {
	/* Enable various levels of assertions. */
	if SDL_ASSERT_LEVEL < 2 {
		SDL_disabled_assert(condition)
	} else {
		SDL_enabled_assert(condition)
	}
}

func SDL_assert_release(condition bool) {
	/* Enable various levels of assertions. */
	if SDL_ASSERT_LEVEL < 3 {
		SDL_disabled_assert(condition)
	} else {
		SDL_enabled_assert(condition)
	}
}

func SDL_assert_paranoid(condition bool) {
	/* Enable various levels of assertions. */
	if SDL_ASSERT_LEVEL < 4 {
		SDL_disabled_assert(condition)
	} else {
		SDL_enabled_assert(condition)
	}
}

/* this assertion is never disabled at any level. */
func SDL_assert_always(condition bool) {
	SDL_enabled_assert(condition)
}

/*
* A callback that fires when an SDL assertion fails.
*
* - data a pointer to the SDL_AssertData structure corresponding to the
*             current assertion
* - userdata what was passed as `userdata` to SDL_SetAssertionHandler()
*
* Returns an SDL_AssertState value indicating how to handle the failure.
 */
type SDL_AssertionHandler func(data *SDL_AssertData, userdata any) SDL_AssertState

var assertionHandler SDL_AssertionHandler = SDL_AssertionHandler(SDL_PromptAssertion)
var assertionData any

/*
 * Set an application-defined assertion handler.
 *
 * This function allows an application to show its own assertion UI and/or
 * force the response to an assertion failure. If the application doesn't
 * provide this, SDL will try to do the right thing, popping up a
 * system-specific GUI dialog, and probably minimizing any fullscreen windows.
 *
 * This callback may fire from any thread, but it runs wrapped in a mutex, so
 * it will only fire from one thread at a time.
 *
 * This callback is NOT reset to SDL's internal handler upon SDL_Quit()!
 *
 * - handler the SDL_AssertionHandler function to call when an assertion
 *                fails or NULL for the default handler
 * - userdata a pointer that is passed to `handler`
 *
 * This function is available since SDL 3.0.0.
 *
 * See also SDL_GetAssertionHandler.
 */
func SDL_SetAssertionHandler(handler SDL_AssertionHandler, userdata any) {
	if handler != nil {
		assertionHandler = handler
		assertionData = userdata
	} else {
		assertionHandler = SDL_PromptAssertion
		assertionData = nil
	}
}

/*
 * Get the default assertion handler.
 *
 * This returns the function pointer that is called by default when an
 * assertion is triggered. This is an internal function provided by SDL, that
 * is used for assertions when SDL_SetAssertionHandler() hasn't been used to
 * provide a different function.
 *
 * Returns the default SDL_AssertionHandler that is called when an assert
 *          triggers.
 *
 * This function is available since SDL 3.0.0.
 *
 * See also SDL_GetAssertionHandler.
 */
func SDL_GetDefaultAssertionHandler() SDL_AssertionHandler {
	return SDL_PromptAssertion
}

/*
 * Get the current assertion handler.
 *
 * This returns the function pointer that is called when an assertion is
 * triggered. This is either the value last passed to
 * SDL_SetAssertionHandler(), or if no application-specified function is set,
 * is equivalent to calling SDL_GetDefaultAssertionHandler().
 *
 * Returns the SDL_AssertionHandler that is called when an assert triggers,
 * and the user data set before.
 *
 * This function is available since SDL 3.0.0.
 *
 * See also SDL_SetAssertionHandler
 */
func SDL_GetAssertionHandler() (SDL_AssertionHandler, any) {
	return assertionHandler, assertionData
}

/**
 * Get a list of all assertion failures.
 *
 * This function gets all assertions triggered since the last call to
 * SDL_ResetAssertionReport(), or the start of the program.
 *
 *
 * Eeturns a list of all failed assertions or NULL if the list is empty. This
 *          memory should not be modified or freed by the application.
 *
 * This function is available since SDL 3.0.0.
 *
 * See also SDL_ResetAssertionReport
 */
func SDL_ResetAssertionReport() {
	var next, item *SDL_AssertData

	for item = triggeredAssertions; item != nil; item = next {
		next = item.Next
		item.AlwaysIgnore = false
		item.TriggerCount = 0
		item.Next = nil
	}

	triggeredAssertions = nil
}

/* The size of the stack buffer to use for rendering assert messages. */
const SDL_MAX_ASSERT_MESSAGE_STACK = 256

/*
 * We keep all triggered assertions in a singly-linked list so we can
 *  generate a report later.
 */
var triggeredAssertions *SDL_AssertData

func debug_print(form string, args ...any) {
	slog.Warn(form, args...)
}

func SDL_AddAssertionToReport(data *SDL_AssertData) {
	data.TriggerCount++
	if data.TriggerCount == 1 { /* not yet added? */
		data.Next = triggeredAssertions
		triggeredAssertions = data
	}
}

const ENDLINE = "\r"

func SDL_RenderAssertMessage(data SDL_AssertData) string {
	return fmt.Sprintf("Assertion failure at %s (%s:%d), triggered %d %s:"+ENDLINE+"  '%s'",
		data.Function, data.Filename, data.Linenum,
		data.TriggerCount, tern((data.TriggerCount == 1), "time", "times"),
		data.Condition)
}

func SDL_GenerateAssertionReport() {
	var item *SDL_AssertData = triggeredAssertions

	if item != nil {
		debug_print("\n\nSDL assertion report.\n")
		debug_print("All SDL assertions between last init/quit:\n\n")

		for item != nil {
			debug_print(
				"'%s'\n"+
					"    * %s (%s:%d)\n"+
					"    * triggered %d time%s.\n"+
					"    * always ignore: %s.\n",
				item.Condition, item.Function, item.Filename,
				item.Linenum, item.TriggerCount,
				tern((item.TriggerCount == 1), "", "s"),
				tern(item.AlwaysIgnore, "yes", "no"))
			item = item.Next
		}
		debug_print("\n")

		SDL_ResetAssertionReport()
	}
}

func SDL_ExitProcess(exitcode int) {
	os.Exit(exitcode)
}

func SDL_AbortAssertion() {
	//  TODO SDL_Quit()
	SDL_ExitProcess(42)
}

func SDL_PromptAssertion(data *SDL_AssertData, userdata any) SDL_AssertState {
	var state SDL_AssertState = SDL_ASSERTION_ABORT
	/*
	   SDL_Window *window;
	   SDL_MessageBoxData messagebox;
	   SDL_MessageBoxButtonData buttons[] = {
	       { 0, SDL_ASSERTION_RETRY, "Retry" },
	       { 0, SDL_ASSERTION_BREAK, "Break" },
	       { 0, SDL_ASSERTION_ABORT, "Abort" },
	       { SDL_MESSAGEBOX_BUTTON_ESCAPEKEY_DEFAULT,
	         SDL_ASSERTION_IGNORE, "Ignore" },
	       { SDL_MESSAGEBOX_BUTTON_RETURNKEY_DEFAULT,
	         SDL_ASSERTION_ALWAYS_IGNORE, "Always Ignore" }
	   };
	   int selected;

	   char stack_buf[SDL_MAX_ASSERT_MESSAGE_STACK];
	   char *message = stack_buf;
	   size_t buf_len = sizeof(stack_buf);
	   int len;

	   (void)userdata; // unused in default handler.

	   //  Assume the output will fit...
	   len = SDL_RenderAssertMessage(message, buf_len, data);

	   // .. and if it didn't, try to allocate as much room as we actually need.
	   if (len >= (int)buf_len) {
	       if (SDL_size_add_overflow(len, 1, &buf_len) == 0) {
	           message = (char *)SDL_malloc(buf_len);
	           if (message) {
	               len = SDL_RenderAssertMessage(message, buf_len, data);
	           } else {
	               message = stack_buf;
	           }
	       }
	   }

	   // Something went very wrong
	   if (len < 0) {
	       if (message != stack_buf) {
	           SDL_free(message);
	       }
	       return SDL_ASSERTION_ABORT;
	   }

	   debug_print("\n\n%s\n\n", message);
	*/

	// let env. variable override, so unit tests won't block in a GUI.
	envr := os.Getenv("SDL_ASSERT")
	if envr != "" {
		if envr == "abort" {
			return SDL_ASSERTION_ABORT
		} else if envr == "break" {
			return SDL_ASSERTION_BREAK
		} else if envr == "retry" {
			return SDL_ASSERTION_RETRY
		} else if envr == "ignore" {
			return SDL_ASSERTION_IGNORE
		} else if envr == "always_ignore" {
			return SDL_ASSERTION_ALWAYS_IGNORE
		} else {
			return SDL_ASSERTION_ABORT /* oh well. */
		}
	}

	/*
		    // Leave fullscreen mode, if possible (scary!)
		    window = SDL_GetToplevelForKeyboardFocus();
		    if (window) {
		        if (window.fullscreen_exclusive) {
		            SDL_MinimizeWindow(window);
		        } else {
		            //* !!! FIXME: ungrab the input if we're not fullscreen?
		            // No need to mess with the window
		            window = NULL;
		        }
		    }

		    // Show a messagebox if we can, otherwise fall back to stdio
		    SDL_zero(messagebox);
		    messagebox.flags = SDL_MESSAGEBOX_WARNING;
		    messagebox.window = window;
		    messagebox.title = "Assertion Failed";
		    messagebox.message = message;
		    messagebox.numbuttons = SDL_arraysize(buttons);
		    messagebox.buttons = buttons;

		    if (SDL_ShowMessageBox(&messagebox, &selected) == 0) {
		        if (selected == -1) {
		            state = SDL_ASSERTION_IGNORE;
		        } else {
		            state = (SDL_AssertState)selected;
		        }
		    } else {
		#ifdef SDL_PLATFORM_EMSCRIPTEN
		        // This is nasty, but we can't block on a custom UI.
		        for (;;) {
		            SDL_bool okay = SDL_TRUE;
		            char *buf = (char *) MAIN_THREAD_EM_ASM_PTR({
		                var str =
		                    UTF8ToString($0) + '\n\n' +
		                    'Abort/Retry/Ignore/AlwaysIgnore? [ariA] :';
		                var reply = window.prompt(str, "i");
		                if (reply === null) {
		                    reply = "i";
		                }
		                return allocate(intArrayFromString(reply), 'i8', ALLOC_NORMAL);
		            }, message);

		            if (SDL_strcmp(buf, "a") == 0) {
		                state = SDL_ASSERTION_ABORT;
		            } else if (SDL_strcmp(buf, "b") == 0) {
		                state = SDL_ASSERTION_BREAK;
		            } else if (SDL_strcmp(buf, "r") == 0) {
		                state = SDL_ASSERTION_RETRY;
		            } else if (SDL_strcmp(buf, "i") == 0) {
		                state = SDL_ASSERTION_IGNORE;
		            } else if (SDL_strcmp(buf, "A") == 0) {
		                state = SDL_ASSERTION_ALWAYS_IGNORE;
		            } else {
		                okay = SDL_FALSE;
		            }
		            free(buf);  // This should NOT be SDL_free()

		            if (okay) {
		                break;
		            }
		        }
	*/
	for {
		var buf string
		fmt.Fprintf(os.Stderr, "Abort/Break/Retry/Ignore/AlwaysIgnore? [abriA] : ")
		os.Stderr.Sync()
		if c, err := fmt.Fscanln(os.Stdin, &buf); c == 0 || err != nil {
			break
		}

		if buf == "a" {
			state = SDL_ASSERTION_ABORT
			break
		} else if buf == "b" {
			state = SDL_ASSERTION_BREAK
			break
		} else if buf == "r" {
			state = SDL_ASSERTION_RETRY
			break
		} else if buf == "i" {
			state = SDL_ASSERTION_IGNORE
			break
		} else if buf == "A" {
			state = SDL_ASSERTION_ALWAYS_IGNORE
			break
		}
	}

	return state
}

func SDL_AssertionsQuit() {
	if SDL_ASSERT_LEVEL > 0 {
		SDL_GenerateAssertionReport()
	}
}
