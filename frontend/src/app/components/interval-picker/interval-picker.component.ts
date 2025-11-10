import { Component, Input, OnInit, OnDestroy, forwardRef } from '@angular/core';
import {
  ControlValueAccessor,
  NG_VALUE_ACCESSOR,
  FormBuilder,
  FormGroup,
  Validators,
} from '@angular/forms';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';
import { CommonModule } from '@angular/common'; // <-- ADD THIS IMPORT
import { ReactiveFormsModule } from '@angular/forms'; // <-- ADD THIS IMPORT

/**
 * A custom form control for picking intervals (e.g., "30m", "2h", "7d").
 * Implements ControlValueAccessor to work with Angular's Reactive Forms.
 */
@Component({
  selector: 'app-interval-picker',
  templateUrl: './interval-picker.component.html',
  styleUrls: ['./interval-picker.component.css'],
  standalone: true, // <-- ADD THIS
  imports: [CommonModule, ReactiveFormsModule], // <-- ADD THIS
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => IntervalPickerComponent),
      multi: true,
    },
  ],
})
export class IntervalPickerComponent
  implements ControlValueAccessor, OnInit, OnDestroy
{
  @Input() readOnly = false;

  public intervalForm: FormGroup;
  private destroy$ = new Subject<void>();

  // ControlValueAccessor callbacks
  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};

  constructor(private fb: FormBuilder) {
    this.intervalForm = this.fb.group({
      value: [10, [Validators.required, Validators.min(1)]], // <-- CHANGED default value to 10
      unit: ['m', Validators.required], // <-- CHANGED default unit to 'm'
    });
  }

  ngOnInit(): void {
    // When the internal form changes, combine the values and propagate the change
    this.intervalForm.valueChanges
      .pipe(takeUntil(this.destroy$))
      .subscribe((val) => {
        if (this.intervalForm.valid) {
          const combinedValue = `${val.value}${val.unit}`;
          this.onChange(combinedValue);
        } else {
          this.onChange(''); // Propagate invalid state
        }
      });
  }

  /**
   * Parses the interval string (e.g., "1h") from the parent form
   * and patches the internal form.
   * @param intervalString The interval string (e.g., "1h", "30m").
   */
  private parseInterval(intervalString: string): void {
    if (!intervalString) {
      this.intervalForm.patchValue(
        { value: 10, unit: 'm' }, // <-- CHANGED default
        { emitEvent: false }
      );
      return;
    }

    // Regex to capture the number and the unit (m, h, or d)
    const match = intervalString.match(/^(\d+)([mhd]?)$/);

    if (match) {
      const value = parseInt(match[1], 10);
      const unit = match[2] || 'm'; // <-- CHANGED default to 'm' if no unit
      this.intervalForm.patchValue({ value, unit }, { emitEvent: false });
    } else {
      // Handle invalid or unexpected format
      this.intervalForm.patchValue(
        { value: 10, unit: 'm' }, // <-- CHANGED default
        { emitEvent: false }
      );
    }
  }

  // --- ControlValueAccessor Implementation ---

  /**
   * Called by the Forms API to write a value to the control.
   * @param value The value from the parent form control.
   */
  writeValue(value: any): void {
    if (typeof value === 'string') {
      this.parseInterval(value);
    }
  }

  /**
   * Registers a callback function to be called when the control's value changes.
   * @param fn The callback function.
   */
  registerOnChange(fn: any): void {
    this.onChange = fn;
  }

  /**
   * Registers a callback function to be called when the control is touched.
   * @param fn The callback function.
   */
  registerOnTouched(fn: any): void {
    this.onTouched = fn;
  }

  /**
   * Called by the Forms API when the control's disabled status changes.
   * @param isDisabled Whether the control should be disabled.
   */
  setDisabledState(isDisabled: boolean): void {
    this.readOnly = isDisabled;
    if (isDisabled) {
      this.intervalForm.disable();
    } else {
      this.intervalForm.enable();
    }
  }

  /**
   * Marks the control as 'touched' when the input is blurred.
   */
  handleBlur(): void {
    this.onTouched();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }
}

