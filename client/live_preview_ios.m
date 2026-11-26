//go:build bundle && cgo

#import <UIKit/UIKit.h>
#import <WebKit/WebKit.h>
#import <arpa/inet.h>
#import <math.h>

static NSInteger previewPort(NSURL *url) {
    if (url.port) return url.port.integerValue;
    return [url.scheme.lowercaseString isEqualToString:@"https"] ? 443 : 80;
}

static UIColor *previewColor(id value) {
    if (![value isKindOfClass:NSString.class] || [value length] != 7 ||
        ![value hasPrefix:@"#"]) return nil;
    unsigned int rgb;
    NSScanner *scanner = [NSScanner scannerWithString:[value substringFromIndex:1]];
    if (![scanner scanHexInt:&rgb] || !scanner.isAtEnd) return nil;
    return [UIColor colorWithRed:((rgb >> 16) & 255) / 255.0
                          green:((rgb >> 8) & 255) / 255.0
                           blue:(rgb & 255) / 255.0 alpha:1];
}

static CGFloat dockNumber(NSDictionary *config, NSString *key, CGFloat fallback, CGFloat minimum, CGFloat maximum) {
    id value = config[key];
    if (![value isKindOfClass:NSNumber.class] || !isfinite([value doubleValue])) return fallback;
    return MIN(maximum, MAX(minimum, [value doubleValue]));
}

@interface BetterMusicLivePreview : UIViewController <WKNavigationDelegate, UITabBarDelegate, UITabBarControllerDelegate>
@property (nonatomic, strong) NSURL *previewURL;
@property (nonatomic, strong) WKWebView *webView;
@property (nonatomic, strong) UITabBar *tabBar;
@property (nonatomic, strong) UITabBarController *systemTabs;
@property (nonatomic, strong) UIView *dockContainer;
@property (nonatomic, strong) NSDictionary *dockConfig;
- (void)selectTabFromPage:(WKScriptMessage *)message;
@end

@interface BetterMusicLivePreviewMessages : NSObject <WKScriptMessageHandler>
@property (nonatomic, weak) BetterMusicLivePreview *preview;
@end

@implementation BetterMusicLivePreviewMessages
- (void)userContentController:(WKUserContentController *)controller
      didReceiveScriptMessage:(WKScriptMessage *)message {
    [self.preview selectTabFromPage:message];
}
@end

@implementation BetterMusicLivePreview
- (void)viewDidLoad {
    [super viewDidLoad];
    self.title = @"Live redesign preview";
    self.view.backgroundColor = UIColor.systemBackgroundColor;
    self.navigationItem.leftBarButtonItem =
        [[UIBarButtonItem alloc] initWithBarButtonSystemItem:UIBarButtonSystemItemDone
                                                    target:self action:@selector(close)];
    self.navigationItem.rightBarButtonItem =
        [[UIBarButtonItem alloc] initWithBarButtonSystemItem:UIBarButtonSystemItemRefresh
                                                    target:self action:@selector(loadPreview)];

    WKWebViewConfiguration *configuration = [[WKWebViewConfiguration alloc] init];
    configuration.websiteDataStore = WKWebsiteDataStore.nonPersistentDataStore;
    BetterMusicLivePreviewMessages *messages = [[BetterMusicLivePreviewMessages alloc] init];
    messages.preview = self;
    [configuration.userContentController addScriptMessageHandler:messages name:@"betterMusicLivePreviewTabs"];
    [configuration.userContentController addScriptMessageHandler:messages name:@"betterMusicLivePreviewDock"];
    self.webView = [[WKWebView alloc] initWithFrame:CGRectZero configuration:configuration];
    self.webView.navigationDelegate = self;

    NSArray<UITabBarItem *> *items = @[
        [[UITabBarItem alloc] initWithTitle:@"Artists" image:[UIImage systemImageNamed:@"person.2"] tag:0],
        [[UITabBarItem alloc] initWithTitle:@"Album detail" image:[UIImage systemImageNamed:@"opticaldisc"] tag:1]
    ];
    if (@available(iOS 26.0, *)) {
        self.systemTabs = [[UITabBarController alloc] init];
        self.systemTabs.delegate = self;
        self.systemTabs.overrideUserInterfaceStyle = UIUserInterfaceStyleDark;
        [self addChildViewController:self.systemTabs];
        self.systemTabs.view.frame = self.view.bounds;
        self.systemTabs.view.autoresizingMask = UIViewAutoresizingFlexibleWidth | UIViewAutoresizingFlexibleHeight;
        [self.view addSubview:self.systemTabs.view];
        [self setSystemTabItems:items];
        [self.systemTabs didMoveToParentViewController:self];
    } else {
        self.tabBar = [[UITabBar alloc] init];
        self.tabBar.delegate = self;
        self.tabBar.items = items;
        self.tabBar.selectedItem = items.firstObject;
        [self.view addSubview:self.webView];
        self.dockContainer = [[UIView alloc] init];
        [self.view addSubview:self.dockContainer];
        [self.dockContainer addSubview:self.tabBar];
    }
    self.dockConfig = @{};
    [self loadPreview];
}

- (void)viewDidLayoutSubviews {
    [super viewDidLayoutSubviews];
    if (self.systemTabs) return;
    NSDictionary *config = self.dockConfig;
    CGRect safe = self.view.safeAreaLayoutGuide.layoutFrame;
    CGFloat margin = dockNumber(config, @"sideMargin", 0, 0, safe.size.width / 3);
    CGFloat available = MAX(0, safe.size.width - 2 * margin);
    CGFloat width = dockNumber(config, @"width", 0, 0, available);
    if (width == 0) width = available;
    CGFloat height = dockNumber(config, @"height", 49, 32, 120);
    CGFloat bottom = dockNumber(config, @"bottomOffset", 0, 0, 120);
    BOOL visible = ![config[@"visible"] isKindOfClass:NSNumber.class] || [config[@"visible"] boolValue];
    CGFloat top = visible ? MAX(CGRectGetMinY(safe), CGRectGetMaxY(safe) - bottom - height) : CGRectGetMaxY(safe);
    self.dockContainer.hidden = !visible;
    self.dockContainer.frame = CGRectMake(CGRectGetMidX(safe) - width / 2, top, width, height);
    self.tabBar.frame = self.dockContainer.bounds;
    self.webView.frame = CGRectMake(safe.origin.x, safe.origin.y, safe.size.width,
                                    MAX(0, top - safe.origin.y));
    CGFloat radius = dockNumber(config, @"cornerRadius", 0, 0, height / 2);
    self.tabBar.layer.cornerRadius = radius;
    self.tabBar.clipsToBounds = YES;
    self.tabBar.layer.borderWidth = dockNumber(config, @"borderWidth", 0, 0, 4);
    self.tabBar.layer.borderColor = (previewColor(config[@"borderColor"]) ?: UIColor.clearColor).CGColor;
    CALayer *shadow = self.dockContainer.layer;
    shadow.shadowColor = (previewColor(config[@"shadowColor"]) ?: UIColor.blackColor).CGColor;
    shadow.shadowOpacity = dockNumber(config, @"shadowOpacity", 0, 0, 1);
    shadow.shadowRadius = dockNumber(config, @"shadowRadius", 12, 0, 40);
    shadow.shadowOffset = CGSizeMake(dockNumber(config, @"shadowX", 0, -40, 40),
                                    dockNumber(config, @"shadowY", 4, -40, 40));
    shadow.shadowPath = [UIBezierPath bezierPathWithRoundedRect:self.dockContainer.bounds cornerRadius:radius].CGPath;
}

- (void)attachWebViewToTab:(UIViewController *)tab {
    UIView *content = tab.view;
    if (self.webView.superview == content) return;
    self.webView.frame = content.bounds;
    self.webView.autoresizingMask = UIViewAutoresizingFlexibleWidth | UIViewAutoresizingFlexibleHeight;
    [content addSubview:self.webView];
}

- (void)setSystemTabItems:(NSArray<UITabBarItem *> *)items {
    NSUInteger selected = self.systemTabs.selectedIndex;
    NSMutableArray<UIViewController *> *controllers = [NSMutableArray arrayWithCapacity:items.count];
    for (UITabBarItem *item in items) {
        UIViewController *tab = [[UIViewController alloc] init];
        tab.tabBarItem = item;
        [controllers addObject:tab];
    }
    [self.systemTabs setViewControllers:controllers animated:NO];
    [self selectPreviewTabAtIndex:selected < controllers.count ? selected : 0];
}

- (void)selectPreviewTabAtIndex:(NSUInteger)index {
    if (self.systemTabs) {
        if (index >= self.systemTabs.viewControllers.count) return;
        self.systemTabs.selectedIndex = index;
        [self attachWebViewToTab:self.systemTabs.selectedViewController];
    } else if (index < self.tabBar.items.count) {
        self.tabBar.selectedItem = self.tabBar.items[index];
    }
}

- (void)loadPreview {
    [self selectPreviewTabAtIndex:0];
    [self.webView loadRequest:[NSURLRequest requestWithURL:self.previewURL
                                             cachePolicy:NSURLRequestReloadIgnoringLocalCacheData
                                         timeoutInterval:15]];
}

- (void)close {
    [self dismissViewControllerAnimated:YES completion:nil];
}

- (BOOL)allowsURL:(NSURL *)url {
    return [url.scheme.lowercaseString isEqualToString:self.previewURL.scheme.lowercaseString] &&
        [url.host.lowercaseString isEqualToString:self.previewURL.host.lowercaseString] &&
        previewPort(url) == previewPort(self.previewURL) &&
        !url.user && !url.password &&
        [url.path isEqualToString:self.previewURL.path];
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationAction:(WKNavigationAction *)action
    decisionHandler:(void (^)(WKNavigationActionPolicy))decide {
    decide(action.targetFrame.isMainFrame && [self allowsURL:action.request.URL]
        ? WKNavigationActionPolicyAllow : WKNavigationActionPolicyCancel);
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationResponse:(WKNavigationResponse *)response
    decisionHandler:(void (^)(WKNavigationResponsePolicy))decide {
    decide(response.isForMainFrame && [self allowsURL:response.response.URL]
        ? WKNavigationResponsePolicyAllow : WKNavigationResponsePolicyCancel);
}

- (void)webView:(WKWebView *)webView didFailProvisionalNavigation:(WKNavigation *)navigation
    withError:(NSError *)error {
    if (error.code == NSURLErrorCancelled || !self.view.window || self.presentedViewController) return;
    UIAlertController *alert = [UIAlertController alertControllerWithTitle:@"Couldn’t load the live preview"
        message:error.localizedDescription
        preferredStyle:UIAlertControllerStyleAlert];
    [alert addAction:[UIAlertAction actionWithTitle:@"OK" style:UIAlertActionStyleDefault handler:nil]];
    [self presentViewController:alert animated:YES completion:nil];
}

- (void)selectTabFromPage:(WKScriptMessage *)message {
    if (!message.frameInfo.isMainFrame || ![self allowsURL:message.frameInfo.request.URL]) return;

    if ([message.name isEqualToString:@"betterMusicLivePreviewDock"]) {
        if (![message.body isKindOfClass:NSDictionary.class]) return;
        NSDictionary *config = message.body;
        NSArray *tabs = config[@"items"];
        if (![tabs isKindOfClass:NSArray.class] || tabs.count < 1 || tabs.count > 5) return;
        NSMutableArray<UITabBarItem *> *items = [NSMutableArray array];
        for (id entry in tabs) {
            if (![entry isKindOfClass:NSDictionary.class]) return;
            NSString *title = entry[@"title"];
            NSString *symbol = entry[@"symbol"];
            if (![title isKindOfClass:NSString.class] || ![symbol isKindOfClass:NSString.class]) return;
            [items addObject:[[UITabBarItem alloc] initWithTitle:title
                image:[UIImage systemImageNamed:symbol] tag:items.count]];
        }
        if (self.systemTabs) {
            if (![self.dockConfig[@"items"] isEqual:tabs]) [self setSystemTabItems:items];
            self.dockConfig = config;
            BOOL hidden = [config[@"visible"] isKindOfClass:NSNumber.class] && ![config[@"visible"] boolValue];
            if (@available(iOS 26.0, *)) [self.systemTabs setTabBarHidden:hidden animated:NO];
            return;
        }
        NSInteger selected = [self.tabBar.items indexOfObject:self.tabBar.selectedItem];
        self.tabBar.items = items;
        self.tabBar.selectedItem = items[selected >= 0 && selected < (NSInteger)items.count ? selected : 0];
        UITabBarAppearance *appearance = [[UITabBarAppearance alloc] init];
        [appearance configureWithDefaultBackground];
        UIColor *background = previewColor(config[@"background"]);
        UIColor *tint = previewColor(config[@"tint"]);
        UIColor *muted = previewColor(config[@"muted"]);
        self.dockConfig = config;
        if (background) self.view.backgroundColor = background;
        appearance.backgroundColor = [(background ?: UIColor.secondarySystemBackgroundColor)
            colorWithAlphaComponent:dockNumber(config, @"backgroundOpacity", 1, 0, 1)];
        NSString *blur = config[@"blur"];
        if ([blur isEqual:@"light"]) appearance.backgroundEffect = [UIBlurEffect effectWithStyle:UIBlurEffectStyleSystemMaterialLight];
        else if ([blur isEqual:@"dark"]) appearance.backgroundEffect = [UIBlurEffect effectWithStyle:UIBlurEffectStyleSystemMaterialDark];
        else if ([blur isEqual:@"system"]) appearance.backgroundEffect = [UIBlurEffect effectWithStyle:UIBlurEffectStyleSystemMaterial];
        else appearance.backgroundEffect = nil;
        appearance.shadowColor = UIColor.clearColor;
        UIFont *font = [UIFont systemFontOfSize:dockNumber(config, @"labelSize", 10, 8, 20)
                                        weight:UIFontWeightMedium];
        for (UITabBarItem *tab in items) {
            CGFloat x = dockNumber(config, @"iconX", 0, -20, 20);
            CGFloat y = dockNumber(config, @"iconY", 0, -20, 20);
            tab.imageInsets = UIEdgeInsetsMake(y, x, -y, -x);
            tab.titlePositionAdjustment = UIOffsetMake(dockNumber(config, @"labelX", 0, -20, 20),
                                                       dockNumber(config, @"labelY", 0, -20, 20));
        }
        for (UITabBarItemAppearance *item in @[appearance.stackedLayoutAppearance,
             appearance.inlineLayoutAppearance, appearance.compactInlineLayoutAppearance]) {
            item.selected.titleTextAttributes = @{NSFontAttributeName:font};
            item.normal.titleTextAttributes = @{NSFontAttributeName:font};
            if (tint) {
                item.selected.iconColor = tint;
                item.selected.titleTextAttributes = @{NSForegroundColorAttributeName:tint, NSFontAttributeName:font};
            }
            if (muted) {
                item.normal.iconColor = muted;
                item.normal.titleTextAttributes = @{NSForegroundColorAttributeName:muted, NSFontAttributeName:font};
            }
        }
        self.tabBar.standardAppearance = appearance;
        self.tabBar.scrollEdgeAppearance = appearance;
        [self.view setNeedsLayout];
        return;
    }
    if (![message.body isKindOfClass:NSNumber.class]) return;

    NSInteger index = [message.body integerValue];
    if (index >= 0) [self selectPreviewTabAtIndex:(NSUInteger)index];
}

- (void)tabBar:(UITabBar *)tabBar didSelectItem:(UITabBarItem *)item {
    [self notifyPageOfTabSelection:item.tag];
}

- (void)tabBarController:(UITabBarController *)tabBarController
didSelectViewController:(UIViewController *)viewController {
    [self attachWebViewToTab:viewController];
    [self notifyPageOfTabSelection:viewController.tabBarItem.tag];
}

- (void)notifyPageOfTabSelection:(NSInteger)index {
    NSString *script = [NSString stringWithFormat:
        @"window.dispatchEvent(new CustomEvent('nativeTabSelected',{detail:{index:%ld}}));", (long)index];
    [self.webView evaluateJavaScript:script completionHandler:nil];
}
@end

void openBetterMusicLivePreview(UIViewController *presenter, NSString *address) {
    NSURLComponents *url = [NSURLComponents componentsWithString:address];
    BOOL tailscaleHTTPS = [url.scheme.lowercaseString isEqualToString:@"https"] &&
        [url.host.lowercaseString hasSuffix:@".ts.net"] && (!url.port || url.port.integerValue == 443);
    struct in_addr addressIPv4;
    BOOL tailscaleIP = [url.scheme.lowercaseString isEqualToString:@"http"] && url.host &&
        inet_pton(AF_INET, url.host.UTF8String, &addressIPv4) == 1 &&
        (ntohl(addressIPv4.s_addr) & 0xffc00000) == 0x64400000 &&
        (!url.port || (url.port.integerValue > 0 && url.port.integerValue <= 65535));
    if ((!tailscaleHTTPS && !tailscaleIP) || url.user || url.password ||
        ![url.path isEqualToString:@"/preview/redesign"] || url.query || url.fragment) return;

    NSURL *previewURL = url.URL;
    if (!previewURL || !presenter.view.window || presenter.presentedViewController) return;
    BetterMusicLivePreview *preview = [[BetterMusicLivePreview alloc] init];
    preview.previewURL = previewURL;
    UINavigationController *navigation = [[UINavigationController alloc] initWithRootViewController:preview];
    navigation.modalPresentationStyle = UIModalPresentationFullScreen;
    [presenter presentViewController:navigation animated:YES completion:nil];
}
