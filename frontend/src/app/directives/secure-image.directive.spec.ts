// frontend/src/app/directives/secure-image.directive.spec.ts
 import { Component, DebugElement } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { By } from '@angular/platform-browser';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { SecureImageDirective } from './secure-image.directive';

// Dummy component to host the directive
@Component({
  template: `<img [secureSrc]="src" (imageError)="onError()">`,
  imports: [SecureImageDirective],
  standalone: true
})
class TestHostComponent {
  src: string | null = '/api/test/image';
  onError = jasmine.createSpy('onError');
}

describe('SecureImageDirective', () => {
  let fixture: ComponentFixture<TestHostComponent>;
  let component: TestHostComponent;
  let httpMock: HttpTestingController;
  let imgEl: DebugElement;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule, SecureImageDirective, TestHostComponent]
    });
    fixture = TestBed.createComponent(TestHostComponent);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    imgEl = fixture.debugElement.query(By.directive(SecureImageDirective));
    fixture.detectChanges();
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should request the image blob and set src on success', () => {
    const req = httpMock.expectOne('/api/test/image');
    expect(req.request.method).toBe('GET');
    expect(req.request.responseType).toBe('blob');

    // Create a dummy blob
    const blob = new Blob(['fake image data'], { type: 'image/jpeg' });
    req.flush(blob);

    fixture.detectChanges();

    // The src should now start with blob:
    expect(imgEl.nativeElement.src).toContain('blob:');
    expect(component.onError).not.toHaveBeenCalled();
  });

  it('should emit imageError and remove loading class on failure', () => {
    const req = httpMock.expectOne('/api/test/image');
    
    // Simulate 404
    // FIX: We must pass a Blob or null as the body because responseType is 'blob'.
    // Passing a string triggers "Automatic conversion to Blob is not supported".
    req.flush(null, { status: 404, statusText: 'Not Found' });

    fixture.detectChanges();

    // Should emit error
    expect(component.onError).toHaveBeenCalled();
    // Should not have loading class
    expect(imgEl.nativeElement.classList.contains('loading-image')).toBeFalse();
  });

  it('should revoke URL when src changes', () => {
    // 1. Initial Load
    const req1 = httpMock.expectOne('/api/test/image');
    req1.flush(new Blob(['data1']));
    fixture.detectChanges();
    
    const url1 = imgEl.nativeElement.src;
    expect(url1).toContain('blob:');

    // Spy on revokeObjectURL
    const revokeSpy = spyOn(URL, 'revokeObjectURL');

    // 2. Change Src
    component.src = '/api/test/image2';
    fixture.detectChanges();

    // 3. New Request
    const req2 = httpMock.expectOne('/api/test/image2');
    
    // FIX: Revocation happens inside the 'next' callback of the subscription,
    // which only runs AFTER the new image is successfully loaded (to prevent flickering).
    req2.flush(new Blob(['data2']));
    fixture.detectChanges();

    // Should trigger revoke of old URL
    expect(revokeSpy).toHaveBeenCalledWith(url1);
  });
});