#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#import <ImageIO/ImageIO.h>
#import <QuickLookThumbnailing/QuickLookThumbnailing.h>
#import <ServiceManagement/ServiceManagement.h>
#include "bridge_darwin.h"

// Callbacks implemented in Go (bridge_darwin.go).
extern void goStatusDropped(const char *pathsJSON);
extern void goDragSessionEnded(bool completed);
extern void goOpenSettings(void);
extern void goGridVisibility(bool visible);
extern void goGridBeak(double x);

static NSStatusItem *statusItem = nil;
static NSWindow *gridWindow = nil;
static NSString *gridWindowTitle = nil;
static EventHotKeyRef hotKeyRef = NULL;
static bool shownForDrag = false;
static bool pinnedMode = false;

void dz_set_pinned(bool pinned) {
    pinnedMode = pinned;
}

static NSWindow *findGridWindow(void) {
    if (gridWindow != nil) {
        return gridWindow;
    }
    for (NSWindow *w in NSApp.windows) {
        if ([w.title isEqualToString:gridWindowTitle]) {
            gridWindow = w;
            break;
        }
    }
    if (gridWindow == nil) {
        gridWindow = NSApp.windows.firstObject;
    }
    if (gridWindow != nil) {
        gridWindow.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
                                        NSWindowCollectionBehaviorFullScreenAuxiliary;
        gridWindow.hidesOnDeactivate = NO;
    }
    return gridWindow;
}

static NSArray<NSString *> *pathsFromJSON(const char *json) {
    if (json == NULL) {
        return @[];
    }
    NSData *data = [NSData dataWithBytes:json length:strlen(json)];
    NSArray *arr = [NSJSONSerialization JSONObjectWithData:data options:0 error:nil];
    return [arr isKindOfClass:NSArray.class] ? arr : @[];
}

static char *jsonFromURLs(NSArray<NSURL *> *urls) {
    NSMutableArray<NSString *> *paths = [NSMutableArray array];
    for (NSURL *u in urls) {
        if (u.isFileURL) {
            [paths addObject:u.path];
        }
    }
    NSData *data = [NSJSONSerialization dataWithJSONObject:paths options:0 error:nil];
    if (data == nil) {
        return strdup("[]");
    }
    NSString *s = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
    return strdup(s.UTF8String);
}

static void showGridInternal(bool activate) {
    NSWindow *win = findGridWindow();
    if (win == nil) {
        return;
    }
    NSRect anchor;
    NSScreen *screen;
    if (statusItem != nil && statusItem.button.window != nil) {
        anchor = statusItem.button.window.frame;
        screen = statusItem.button.window.screen ?: NSScreen.mainScreen;
    } else {
        screen = NSScreen.mainScreen;
        anchor = NSMakeRect(NSMidX(screen.frame), NSMaxY(screen.visibleFrame), 0, 0);
    }
    CGFloat x = NSMidX(anchor) - win.frame.size.width / 2.0;
    NSRect visible = screen.visibleFrame;
    x = MAX(NSMinX(visible) + 8, MIN(x, NSMaxX(visible) - win.frame.size.width - 8));
    [win setFrameTopLeftPoint:NSMakePoint(x, NSMinY(anchor) - 2)];
    [win makeKeyAndOrderFront:nil];
    if (activate) {
        [NSApp activateIgnoringOtherApps:YES];
    }
    // Tell the frontend where the status icon sits relative to the window so
    // the popover beak can point at it (the window may be clamped at screen
    // edges).
    goGridBeak(NSMidX(anchor) - x);
    statusItem.button.highlighted = YES;
    goGridVisibility(true);
}

void dz_show_grid(bool activate) {
    dispatch_async(dispatch_get_main_queue(), ^{
        showGridInternal(activate);
    });
}

void dz_hide_grid(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *win = findGridWindow();
        if (win != nil && win.isVisible) {
            [win orderOut:nil];
            statusItem.button.highlighted = NO;
            goGridVisibility(false);
        }
    });
}

void dz_toggle_grid(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *win = findGridWindow();
        if (win != nil && win.isVisible) {
            [win orderOut:nil];
            statusItem.button.highlighted = NO;
            goGridVisibility(false);
        } else {
            showGridInternal(true);
        }
    });
}

bool dz_grid_visible(void) {
    NSWindow *win = gridWindow;
    return win != nil && win.isVisible;
}

// --- Status item -------------------------------------------------------

// Overlay view on the status button: handles clicks, right-click menu, and
// acts as the drag destination for files dragged onto the menu bar icon.
@interface DZStatusView : NSView
@end

@implementation DZStatusView

- (void)mouseUp:(NSEvent *)event {
    dz_toggle_grid();
}

- (void)rightMouseUp:(NSEvent *)event {
    NSMenu *menu = [[NSMenu alloc] init];
    NSMenuItem *settings = [menu addItemWithTitle:@"Settings…"
                                           action:@selector(openSettings:)
                                    keyEquivalent:@""];
    settings.target = self;
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quit = [menu addItemWithTitle:@"Quit DragZone"
                                       action:@selector(quitApp:)
                                keyEquivalent:@""];
    quit.target = self;
    [NSMenu popUpContextMenu:menu withEvent:event forView:self];
}

- (void)openSettings:(id)sender {
    dz_show_grid(true);
    goOpenSettings();
}

- (void)quitApp:(id)sender {
    [NSApp terminate:nil];
}

- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)sender {
    dz_show_grid(false);
    return NSDragOperationCopy;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)sender {
    NSArray<NSURL *> *urls =
        [sender.draggingPasteboard readObjectsForClasses:@[ NSURL.class ]
                                                 options:@{NSPasteboardURLReadingFileURLsOnlyKey : @YES}];
    if (urls.count == 0) {
        return NO;
    }
    char *json = jsonFromURLs(urls);
    goStatusDropped(json);
    free(json);
    return YES;
}

@end

// --- Drag-out source ----------------------------------------------------

@interface DZDragSource : NSObject <NSDraggingSource>
@end

@implementation DZDragSource

- (NSDragOperation)draggingSession:(NSDraggingSession *)session
    sourceOperationMaskForDraggingContext:(NSDraggingContext)context {
    return NSDragOperationCopy | NSDragOperationMove | NSDragOperationGeneric;
}

- (void)draggingSession:(NSDraggingSession *)session
           endedAtPoint:(NSPoint)screenPoint
              operation:(NSDragOperation)operation {
    goDragSessionEnded(operation != NSDragOperationNone);
}

@end

static DZDragSource *dragSource = nil;

void dz_start_drag(const char *pathsJSON) {
    NSArray<NSString *> *paths = pathsFromJSON(pathsJSON);
    if (paths.count == 0) {
        return;
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *win = findGridWindow();
        NSEvent *event = NSApp.currentEvent;
        if (win == nil || event == nil || event.window != win) {
            goDragSessionEnded(false);
            return;
        }
        NSView *view = win.contentView;
        NSPoint location = [view convertPoint:event.locationInWindow fromView:nil];
        NSMutableArray<NSDraggingItem *> *items = [NSMutableArray array];
        CGFloat offset = 0;
        for (NSString *path in paths) {
            NSURL *url = [NSURL fileURLWithPath:path];
            NSDraggingItem *item = [[NSDraggingItem alloc] initWithPasteboardWriter:url];
            NSImage *icon = [NSWorkspace.sharedWorkspace iconForFile:path];
            NSRect frame = NSMakeRect(location.x - 24 + offset, location.y - 24, 48, 48);
            [item setDraggingFrame:frame contents:icon];
            [items addObject:item];
            offset += 6;
        }
        if (dragSource == nil) {
            dragSource = [[DZDragSource alloc] init];
        }
        [view beginDraggingSessionWithItems:items event:event source:dragSource];
    });
}

// --- AirDrop ------------------------------------------------------------

int dz_airdrop(const char *pathsJSON) {
    NSArray<NSString *> *paths = pathsFromJSON(pathsJSON);
    if (paths.count == 0) {
        return 1;
    }
    NSMutableArray<NSURL *> *urls = [NSMutableArray array];
    for (NSString *p in paths) {
        [urls addObject:[NSURL fileURLWithPath:p]];
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        NSSharingService *svc =
            [NSSharingService sharingServiceNamed:NSSharingServiceNameSendViaAirDrop];
        if (svc != nil && [svc canPerformWithItems:urls]) {
            [svc performWithItems:urls];
        }
    });
    return 0;
}

// --- File icons & thumbnails ---------------------------------------------

// pngBase64FromImage renders an NSImage into a base64 PNG at up to
// size*size points, preserving aspect ratio. Returns malloc'd string or NULL.
static char *pngBase64FromImage(NSImage *image, int size) {
    if (image == nil || image.size.width <= 0 || image.size.height <= 0) {
        return NULL;
    }
    CGFloat scale = MIN(size / image.size.width, size / image.size.height);
    NSSize target = NSMakeSize(image.size.width * scale, image.size.height * scale);
    NSImage *resized = [[NSImage alloc] initWithSize:target];
    [resized lockFocus];
    [image drawInRect:NSMakeRect(0, 0, target.width, target.height)
             fromRect:NSZeroRect
            operation:NSCompositingOperationCopy
             fraction:1.0];
    [resized unlockFocus];
    CGImageRef cg = [resized CGImageForProposedRect:NULL context:nil hints:nil];
    if (cg == NULL) {
        return NULL;
    }
    NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithCGImage:cg];
    rep.size = target;
    NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
    if (png == nil) {
        return NULL;
    }
    return strdup([png base64EncodedStringWithOptions:0].UTF8String);
}

char *dz_file_thumbnail_png_base64(const char *cpath, int size) {
    if (@available(macOS 10.15, *)) {
        NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:cpath]];
        QLThumbnailGenerationRequest *req = [[QLThumbnailGenerationRequest alloc]
            initWithFileAtURL:url
                         size:CGSizeMake(size, size)
                        scale:2.0
          representationTypes:QLThumbnailGenerationRequestRepresentationTypeThumbnail];
        dispatch_semaphore_t sem = dispatch_semaphore_create(0);
        __block char *result = NULL;
        [QLThumbnailGenerator.sharedGenerator
            generateBestRepresentationForRequest:req
                               completionHandler:^(QLThumbnailRepresentation *thumb, NSError *error) {
                                   if (thumb != nil) {
                                       result = pngBase64FromImage(thumb.NSImage, size);
                                   }
                                   dispatch_semaphore_signal(sem);
                               }];
        // Bounded wait: thumbnailing a huge video must not hang a drop.
        dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, (int64_t)(2 * NSEC_PER_SEC)));
        return result;
    }
    return NULL;
}

char *dz_file_icon_png_base64(const char *cpath, int size) {
    NSString *path = [NSString stringWithUTF8String:cpath];
    __block char *result = NULL;
    void (^render)(void) = ^{
        NSImage *icon = [NSWorkspace.sharedWorkspace iconForFile:path];
        NSRect rect = NSMakeRect(0, 0, size, size);
        NSImage *resized = [[NSImage alloc] initWithSize:rect.size];
        [resized lockFocus];
        [icon drawInRect:rect
                fromRect:NSZeroRect
               operation:NSCompositingOperationCopy
                fraction:1.0];
        [resized unlockFocus];
        CGImageRef cg = [resized CGImageForProposedRect:NULL context:nil hints:nil];
        if (cg != NULL) {
            NSBitmapImageRep *rep = [[NSBitmapImageRep alloc] initWithCGImage:cg];
            rep.size = rect.size;
            NSData *png = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
            if (png != nil) {
                result = strdup([png base64EncodedStringWithOptions:0].UTF8String);
            }
        }
    };
    if (NSThread.isMainThread) {
        render();
    } else {
        dispatch_sync(dispatch_get_main_queue(), render);
    }
    return result;
}

// --- Login item ---------------------------------------------------------

int dz_set_login_item(bool enabled) {
    if (@available(macOS 13.0, *)) {
        NSError *err = nil;
        SMAppService *svc = [SMAppService mainAppService];
        if (enabled) {
            return [svc registerAndReturnError:&err] ? 0 : -2;
        }
        return [svc unregisterAndReturnError:&err] ? 0 : -2;
    }
    return -1;
}

// --- Global hotkey ------------------------------------------------------

static OSStatus hotKeyHandler(EventHandlerCallRef next, EventRef event, void *userData) {
    dz_toggle_grid();
    return noErr;
}

void dz_set_hotkey_f(int fkey) {
    dispatch_async(dispatch_get_main_queue(), ^{
        static bool handlerInstalled = false;
        if (hotKeyRef != NULL) {
            UnregisterEventHotKey(hotKeyRef);
            hotKeyRef = NULL;
        }
        if (fkey <= 0 || fkey > 12) {
            return;
        }
        if (!handlerInstalled) {
            EventTypeSpec spec = {kEventClassKeyboard, kEventHotKeyPressed};
            InstallApplicationEventHandler(&hotKeyHandler, 1, &spec, NULL, NULL);
            handlerInstalled = true;
        }
        static const UInt32 codes[13] = {0,   kVK_F1, kVK_F2, kVK_F3, kVK_F4,
                                         kVK_F5, kVK_F6, kVK_F7, kVK_F8, kVK_F9,
                                         kVK_F10, kVK_F11, kVK_F12};
        EventHotKeyID hkid = {.signature = 'dzhk', .id = 1};
        RegisterEventHotKey(codes[fkey], 0, hkid, GetApplicationEventTarget(), 0, &hotKeyRef);
    });
}

bool dz_option_key_down(void) {
    return (NSEvent.modifierFlags & NSEventModifierFlagOption) != 0;
}

int dz_strip_image_metadata(const char *csrc, const char *cdst) {
    NSURL *src = [NSURL fileURLWithPath:[NSString stringWithUTF8String:csrc]];
    NSURL *dst = [NSURL fileURLWithPath:[NSString stringWithUTF8String:cdst]];
    CGImageSourceRef source = CGImageSourceCreateWithURL((__bridge CFURLRef)src, NULL);
    if (source == NULL) {
        return -1;
    }
    CFStringRef type = CGImageSourceGetType(source);
    size_t count = CGImageSourceGetCount(source);
    CGImageDestinationRef dest =
        CGImageDestinationCreateWithURL((__bridge CFURLRef)dst, type, count, NULL);
    if (dest == NULL) {
        CFRelease(source);
        return -2;
    }
    NSDictionary *stripped = @{
        (id)kCGImagePropertyExifDictionary : (id)kCFNull,
        (id)kCGImagePropertyExifAuxDictionary : (id)kCFNull,
        (id)kCGImagePropertyGPSDictionary : (id)kCFNull,
        (id)kCGImagePropertyTIFFDictionary : (id)kCFNull,
        (id)kCGImagePropertyIPTCDictionary : (id)kCFNull,
        (id)kCGImagePropertyMakerAppleDictionary : (id)kCFNull,
    };
    for (size_t i = 0; i < count; i++) {
        CGImageDestinationAddImageFromSource(dest, source, i,
                                             (__bridge CFDictionaryRef)stripped);
    }
    bool ok = CGImageDestinationFinalize(dest);
    CFRelease(dest);
    CFRelease(source);
    return ok ? 0 : -3;
}

// --- Init ---------------------------------------------------------------

void dz_init(const char *windowTitle) {
    gridWindowTitle = [NSString stringWithUTF8String:windowTitle];
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];

        NSWindow *win = findGridWindow();
        if (win != nil && win.isVisible) {
            [win orderOut:nil];
        }

        statusItem = [NSStatusBar.systemStatusBar statusItemWithLength:NSSquareStatusItemLength];
        NSImage *img = [NSImage imageWithSystemSymbolName:@"square.grid.3x3.topleft.filled"
                                 accessibilityDescription:@"DragZone"];
        img.template = YES;
        statusItem.button.image = img;

        DZStatusView *overlay = [[DZStatusView alloc] initWithFrame:statusItem.button.bounds];
        overlay.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
        [overlay registerForDraggedTypes:@[ NSPasteboardTypeFileURL ]];
        [statusItem.button addSubview:overlay];

        // Hide the grid when the app deactivates (click elsewhere), except
        // while it is being shown passively for an in-flight drag.
        [NSNotificationCenter.defaultCenter
            addObserverForName:NSApplicationDidResignActiveNotification
                        object:nil
                         queue:NSOperationQueue.mainQueue
                    usingBlock:^(NSNotification *note) {
                        if (!shownForDrag && !pinnedMode) {
                            dz_hide_grid();
                        }
                    }];

        // Show the grid when a file drag reaches the menu bar area.
        [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskLeftMouseDragged
                                               handler:^(NSEvent *e) {
            NSPasteboard *pb = [NSPasteboard pasteboardWithName:NSPasteboardNameDrag];
            if ([pb availableTypeFromArray:@[ NSPasteboardTypeFileURL ]] == nil) {
                return;
            }
            NSScreen *screen = statusItem.button.window.screen ?: NSScreen.mainScreen;
            if (NSEvent.mouseLocation.y >= NSMaxY(screen.frame) - 40 && !dz_grid_visible()) {
                shownForDrag = true;
                dz_show_grid(false);
            }
        }];
        [NSEvent addGlobalMonitorForEventsMatchingMask:NSEventMaskLeftMouseUp
                                               handler:^(NSEvent *e) {
            if (!shownForDrag) {
                return;
            }
            shownForDrag = false;
            dispatch_after(dispatch_time(DISPATCH_TIME_NOW, (int64_t)(0.4 * NSEC_PER_SEC)),
                           dispatch_get_main_queue(), ^{
                NSWindow *win = findGridWindow();
                if (win != nil && win.isVisible &&
                    !NSPointInRect(NSEvent.mouseLocation, win.frame)) {
                    dz_hide_grid();
                }
            });
        }];
    });
}
