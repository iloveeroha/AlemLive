import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/features/reports/presentation/widgets/report_video_placeholder.dart';
import 'package:flutter/material.dart';
import 'package:video_player/video_player.dart';

class ReportVideoPlayer extends StatelessWidget {
  const ReportVideoPlayer({
    required this.recordingUrl,
    required this.controller,
    required this.initialization,
    required this.error,
    required this.onTogglePlayback,
    required this.onRetry,
    super.key,
  });

  final String? recordingUrl;
  final VideoPlayerController? controller;
  final Future<void>? initialization;
  final Object? error;
  final VoidCallback onTogglePlayback;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    if (recordingUrl == null || recordingUrl!.trim().isEmpty) {
      return const ReportVideoPlaceholder();
    }

    if (error != null) {
      return _VideoFrame(
        child: _VideoMessage(
          icon: Icons.error_outline_rounded,
          title: 'Видео не загрузилось',
          actionLabel: 'Повторить',
          onAction: onRetry,
        ),
      );
    }

    final videoController = controller;
    final initFuture = initialization;
    if (videoController == null || initFuture == null) {
      return const _VideoFrame(
        child: Center(child: CircularProgressIndicator(color: Colors.white)),
      );
    }

    return FutureBuilder<void>(
      future: initFuture,
      builder: (context, snapshot) {
        if (snapshot.connectionState != ConnectionState.done) {
          return const _VideoFrame(
            child: Center(
              child: CircularProgressIndicator(color: Colors.white),
            ),
          );
        }

        if (snapshot.hasError || !videoController.value.isInitialized) {
          return _VideoFrame(
            child: _VideoMessage(
              icon: Icons.error_outline_rounded,
              title: 'Видео не загрузилось',
              actionLabel: 'Повторить',
              onAction: onRetry,
            ),
          );
        }

        return AspectRatio(
          aspectRatio: videoController.value.aspectRatio,
          child: ClipRRect(
            borderRadius: BorderRadius.circular(18),
            child: Stack(
              fit: StackFit.expand,
              children: [
                ColoredBox(
                  color: Colors.black,
                  child: VideoPlayer(videoController),
                ),
                AnimatedBuilder(
                  animation: videoController,
                  builder: (context, _) {
                    final isPlaying = videoController.value.isPlaying;
                    return Stack(
                      children: [
                        Center(
                          child: _PlayButton(
                            isPlaying: isPlaying,
                            onPressed: onTogglePlayback,
                          ),
                        ),
                        Positioned(
                          left: 14,
                          right: 14,
                          bottom: 12,
                          child: VideoProgressIndicator(
                            videoController,
                            allowScrubbing: true,
                            colors: const VideoProgressColors(
                              playedColor: AppTheme.blue,
                              bufferedColor: Color(0x88FFFFFF),
                              backgroundColor: Color(0x44FFFFFF),
                            ),
                          ),
                        ),
                      ],
                    );
                  },
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

class _VideoFrame extends StatelessWidget {
  const _VideoFrame({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 16 / 9,
      child: Container(
        decoration: BoxDecoration(
          color: const Color(0xFF101828),
          borderRadius: BorderRadius.circular(18),
        ),
        child: child,
      ),
    );
  }
}

class _VideoMessage extends StatelessWidget {
  const _VideoMessage({
    required this.icon,
    required this.title,
    required this.actionLabel,
    required this.onAction,
  });

  final IconData icon;
  final String title;
  final String actionLabel;
  final VoidCallback onAction;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, color: Colors.white, size: 36),
          const SizedBox(height: 10),
          Text(
            title,
            style: const TextStyle(
              color: Colors.white,
              fontWeight: FontWeight.w800,
            ),
          ),
          const SizedBox(height: 10),
          TextButton(onPressed: onAction, child: Text(actionLabel)),
        ],
      ),
    );
  }
}

class _PlayButton extends StatelessWidget {
  const _PlayButton({required this.isPlaying, required this.onPressed});

  final bool isPlaying;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return IconButton.filled(
      style: IconButton.styleFrom(
        backgroundColor: Colors.white.withValues(alpha: 0.18),
        foregroundColor: Colors.white,
        fixedSize: const Size(58, 58),
      ),
      onPressed: onPressed,
      icon: Icon(
        isPlaying ? Icons.pause_rounded : Icons.play_arrow_rounded,
        size: 38,
      ),
    );
  }
}
