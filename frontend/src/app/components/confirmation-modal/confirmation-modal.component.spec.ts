// frontend/src/app/components/confirmation-modal/confirmation-modal.component.spec.ts

import { ComponentFixture, TestBed } from '@angular/core';
import { Subject } from 'rxjs';
import { ConfirmationModalComponent, ConfirmationModalData } from './confirmation-modal.component';
import { ModalService, ModalEvent } from '../../services/modal.service';
import { ChangeDetectionStrategy, Component, Input } from '@angular/core';

// Mock the app-modal component to avoid its complexities in this test
@Component({ selector: 'app-modal', template: '<ng-content></ng-content>' })
class MockModalComponent {
  @Input() modalId: string = '';
  @Input() modalTitle: string = '';
}

describe('ConfirmationModalComponent', () => {
  let component: ConfirmationModalComponent;
  let fixture: ComponentFixture<ConfirmationModalComponent>;
  let mockModalService: jasmine.SpyObj<ModalService>;
  let modalEventSubject: Subject<ModalEvent>;

  const mockModalData: ConfirmationModalData = {
    title: 'Test Confirmation',
    message: 'Are you sure?',
  };

  beforeEach(async () => {
    modalEventSubject = new Subject<ModalEvent>();
    const modalServiceSpy = jasmine.createSpyObj('ModalService', ['close', 'getModalEvents']);
    modalServiceSpy.getModalEvents.and.returnValue(modalEventSubject.asObservable());

    await TestBed.configureTestingModule({
      declarations: [ ConfirmationModalComponent, MockModalComponent ],
      providers: [
        { provide: ModalService, useValue: modalServiceSpy },
      ],
    })
    .overrideComponent(ConfirmationModalComponent, {
        set: { changeDetection: ChangeDetectionStrategy.Default }
    })
    .compileComponents();

    fixture = TestBed.createComponent(ConfirmationModalComponent);
    component = fixture.componentInstance;
    mockModalService = TestBed.inject(ModalService) as jasmine.SpyObj<ModalService>;

    fixture.detectChanges(); // ngOnInit subscribes
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should display the message when opened with data', () => {
    modalEventSubject.next({ id: ConfirmationModalComponent.MODAL_ID, action: 'open', data: mockModalData });
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('p')?.textContent).toBe('Are you sure?');
  });

  it('should call modalService.close(true) on confirm', () => {
    modalEventSubject.next({ id: ConfirmationModalComponent.MODAL_ID, action: 'open', data: mockModalData });
    fixture.detectChanges();

    const confirmButton = fixture.nativeElement.querySelector('.btn-confirm') as HTMLButtonElement;
    confirmButton.click();

    expect(mockModalService.close).toHaveBeenCalledWith(true);
  });

  it('should call modalService.close(false) on cancel', () => {
    modalEventSubject.next({ id: ConfirmationModalComponent.MODAL_ID, action: 'open', data: mockModalData });
    fixture.detectChanges();

    const cancelButton = fixture.nativeElement.querySelector('.btn-cancel') as HTMLButtonElement;
    cancelButton.click();

    expect(mockModalService.close).toHaveBeenCalledWith(false);
  });
});

