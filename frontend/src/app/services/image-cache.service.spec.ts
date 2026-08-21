// frontend/src/app/services/image-cache.service.spec.ts
import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { ImageCacheService } from './image-cache.service';

describe('ImageCacheService', () => {
  let service: ImageCacheService;
  let httpMock: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [ImageCacheService]
    });
    service = TestBed.inject(ImageCacheService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('should fetch blob and return object URL', (done) => {
    const testUrl = '/api/media/1/preview';
    const fakeBlob = new Blob(['image-bytes'], { type: 'image/jpeg' });

    service.getBlobUrl(testUrl).subscribe(blobUrl => {
      expect(blobUrl).toContain('blob:');
      done();
    });

    const req = httpMock.expectOne(testUrl);
    expect(req.request.method).toBe('GET');
    expect(req.request.responseType).toBe('blob');
    req.flush(fakeBlob);
  });

  it('should share the same Observable and blob URL for identical URL requests', (done) => {
    const testUrl = '/api/media/1/preview';
    const fakeBlob = new Blob(['image-bytes'], { type: 'image/jpeg' });

    let firstResult = '';
    let secondResult = '';

    service.getBlobUrl(testUrl).subscribe(url => {
      firstResult = url;
    });

    service.getBlobUrl(testUrl).subscribe(url => {
      secondResult = url;
      expect(secondResult).toBe(firstResult);
      expect(secondResult).toContain('blob:');
      done();
    });

    const req = httpMock.expectOne(testUrl);
    req.flush(fakeBlob);
  });

  it('should revoke blob URL on invalidate', (done) => {
    const testUrl = '/api/media/1/preview';
    const fakeBlob = new Blob(['image-bytes'], { type: 'image/jpeg' });
    const revokeSpy = spyOn(URL, 'revokeObjectURL');

    service.getBlobUrl(testUrl).subscribe(blobUrl => {
      service.invalidate(testUrl);
      expect(revokeSpy).toHaveBeenCalledWith(blobUrl);
      done();
    });

    const req = httpMock.expectOne(testUrl);
    req.flush(fakeBlob);
  });

  it('should revoke all blob URLs on clearAll', (done) => {
    const testUrl1 = '/api/media/1/preview';
    const testUrl2 = '/api/media/2/preview';
    const fakeBlob1 = new Blob(['image-bytes-1'], { type: 'image/jpeg' });
    const fakeBlob2 = new Blob(['image-bytes-2'], { type: 'image/jpeg' });
    const revokeSpy = spyOn(URL, 'revokeObjectURL');

    let url1 = '';
    let url2 = '';

    service.getBlobUrl(testUrl1).subscribe(bUrl => {
      url1 = bUrl;
      service.getBlobUrl(testUrl2).subscribe(bUrl2 => {
        url2 = bUrl2;
        service.clearAll();
        expect(revokeSpy).toHaveBeenCalledWith(url1);
        expect(revokeSpy).toHaveBeenCalledWith(url2);
        done();
      });
    });

    httpMock.expectOne(testUrl1).flush(fakeBlob1);
    httpMock.expectOne(testUrl2).flush(fakeBlob2);
  });

  it('should remove from cache on HTTP error', (done) => {
    const testUrl = '/api/media/invalid/preview';

    service.getBlobUrl(testUrl).subscribe({
      next: () => fail('Should have failed with 404'),
      error: (err) => {
        expect(err.status).toBe(404);
        
        // Next request should attempt a new HTTP call
        service.getBlobUrl(testUrl).subscribe({
          next: () => fail('Should have failed with 404 again'),
          error: (err2) => {
            expect(err2.status).toBe(404);
            done();
          }
        });

        const req2 = httpMock.expectOne(testUrl);
        req2.flush(null, { status: 404, statusText: 'Not Found' });
      }
    });

    const req1 = httpMock.expectOne(testUrl);
    req1.flush(null, { status: 404, statusText: 'Not Found' });
  });
});
