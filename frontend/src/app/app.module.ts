// frontend/src/app/app.module.ts

import { NgModule } from '@angular/core';
import { BrowserModule } from '@angular/platform-browser';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http'; // <-- Imported HTTP_INTERCEPTORS
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { CommonModule } from '@angular/common';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';

// Import all your components
import { LoginPageComponent } from './pages/login-page/login-page.component';
import { DashboardPageComponent } from './pages/dashboard-page/dashboard-page.component';
import { PageNotFoundComponent } from './pages/page-not-found/page-not-found.component';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { EntryListComponent } from './components/entry-list/entry-list.component';
import { DatabaseSettingsComponent } from './components/database-settings/database-settings.component';
import { NotificationHostComponent } from './components/notification-host/notification-host.component';
import { ModalComponent } from './components/modal/modal.component';
import { CreateDatabaseModalComponent } from './components/create-database-modal/create-database-modal.component';
import { UploadEntryModalComponent } from './components/upload-entry-modal/upload-entry-modal.component';
import { EntryDetailModalComponent } from './components/entry-detail-modal/entry-detail-modal.component';
import { EditEntryModalComponent } from './components/edit-entry-modal/edit-entry-modal.component';
import { ConfirmationModalComponent } from './components/confirmation-modal/confirmation-modal.component';
import { AdminUserListComponent } from './components/admin-user-list/admin-user-list.component';
import { ChangePasswordModalComponent } from './components/change-password-modal/change-password-modal.component';
import { UserFormComponent } from './components/user-form/user-form.component';
import { IntervalPickerComponent } from './components/interval-picker/interval-picker.component';
import { EntryGridComponent } from './components/entry-grid/entry-grid.component';
import { EntryListViewComponent } from './components/entry-list-view/entry-list-view.component';

// Pipes
import { SecureImagePipe } from './pipes/secure-image-pipe';
import { FormatBytesPipe } from './pipes/format-bytes.pipe';

// NEW: Import the JWT Interceptor
import { JwtInterceptor } from './interceptors/jwt.interceptor';

@NgModule({
  declarations: [
    AppComponent,
    LoginPageComponent,
    DashboardPageComponent,
    SidebarComponent,
    EntryListComponent,
    DatabaseSettingsComponent,
    PageNotFoundComponent,
    NotificationHostComponent,
    ModalComponent,
    CreateDatabaseModalComponent,
    UploadEntryModalComponent,
    EntryDetailModalComponent,
    EditEntryModalComponent,
    ConfirmationModalComponent,
    AdminUserListComponent,
    ChangePasswordModalComponent,
    UserFormComponent,
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    HttpClientModule,
    FormsModule,
    ReactiveFormsModule,
    BrowserAnimationsModule,
    CommonModule, 
    IntervalPickerComponent, 
    EntryGridComponent,
    EntryListViewComponent,
    SecureImagePipe,
    FormatBytesPipe, 
  ],
  providers: [
    // NEW: Register the JWT Interceptor
    { provide: HTTP_INTERCEPTORS, useClass: JwtInterceptor, multi: true }
  ],
  bootstrap: [AppComponent],
})
export class AppModule {}