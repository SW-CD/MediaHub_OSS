// frontend/src/app/directives/datetime-default.directive.ts
import { Directive, HostListener, Optional, Self } from '@angular/core';
import { NgControl } from '@angular/forms';

@Directive({
  selector: '[appDatetimeDefault]',
  standalone: true
})
export class DatetimeDefaultDirective {

  // Inject NgControl to cleanly update Reactive Forms
  constructor(@Optional() @Self() private ngControl: NgControl) {}

  // Automatically listen to click and focus events on the host element
  @HostListener('click')
  @HostListener('focus')
  onInteraction(): void {
    // Only proceed if the control exists and is currently empty
    if (this.ngControl && this.ngControl.control && !this.ngControl.value) {
      const now = new Date();
      const year = now.getFullYear();
      
      // Pad single digits with leading zeros
      const month = String(now.getMonth() + 1).padStart(2, '0');
      const day = String(now.getDate()).padStart(2, '0');
      
      const defaultDateTime = `${year}-${month}-${day}T00:00`;
      
      // Update the form control value programmatically
      this.ngControl.control.setValue(defaultDateTime);
    }
  }
}