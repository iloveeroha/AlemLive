import 'package:alem_live_mobile/features/reports/data/models/action_item_model.dart';
import 'package:alem_live_mobile/features/reports/data/models/speaker_activity_model.dart';
import 'package:alem_live_mobile/features/reports/data/models/transcript_segment_model.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/action_item.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';

class ReportModel extends Report {
  const ReportModel({
    required super.id,
    required super.roomName,
    required super.startedAt,
    required super.duration,
    required super.status,
    required super.summary,
    required super.topics,
    required super.takeaways,
    required super.actionItems,
    required super.speakerActivity,
    required super.transcript,
    super.recordingUrl,
  });

  factory ReportModel.fromJson(
    Map<String, dynamic> json, {
    String? backendBaseUrl,
  }) {
    final root = _asMap(json['report']) ?? json;
    final id = _readString(root['id'], fallback: 'report');
    final actionItems = _mapsFrom(
      json['actionItems'] ?? root['actionItems'],
    ).map(ActionItemModel.fromJson).toList();
    final speakerActivity = _markMostActive(
      _mapsFrom(
        json['speakerStats'] ?? root['speakerActivity'],
      ).map(SpeakerActivityModel.fromJson).toList(),
    );
    final transcript = _mapsFrom(
      json['transcriptLines'] ?? json['transcript'] ?? root['transcript'],
    ).map(TranscriptSegmentModel.fromJson).toList();

    return ReportModel(
      id: id,
      roomName: _readString(
        json['roomName'] ?? root['roomName'] ?? root['title'],
        fallback: 'AlemLive',
      ),
      startedAt: _readDateTime(
        root['startedAt'] ?? root['createdAt'] ?? root['date'],
      ),
      duration: _readDuration(root['durationSeconds'] ?? root['duration']),
      status: _readStatus(root['status'] ?? root['processingState']),
      summary: _readSummary(json, root),
      topics: _readTopics(json, root),
      takeaways: _readTakeaways(json, root),
      actionItems: actionItems,
      speakerActivity: speakerActivity,
      transcript: transcript,
      recordingUrl: _readRecordingUrl(
        json: json,
        root: root,
        reportId: id,
        backendBaseUrl: backendBaseUrl,
      ),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'roomName': roomName,
      'startedAt': startedAt.toIso8601String(),
      'durationSeconds': duration.inSeconds,
      'status': status.name,
      'summary': summary,
      'topics': topics,
      'takeaways': takeaways,
      'actionItems': actionItems
          .map(
            (item) => ActionItemModel(
              task: item.task,
              owner: item.owner,
              dueDate: item.dueDate,
              status: item.status,
            ).toJson(),
          )
          .toList(),
      'speakerActivity': speakerActivity
          .map(
            (activity) => SpeakerActivityModel(
              speakerName: activity.speakerName,
              talkTime: activity.talkTime,
              participationPercent: activity.participationPercent,
              isMostActive: activity.isMostActive,
            ).toJson(),
          )
          .toList(),
      'transcript': transcript
          .map(
            (segment) => TranscriptSegmentModel(
              timecode: segment.timecode,
              speakerName: segment.speakerName,
              text: segment.text,
            ).toJson(),
          )
          .toList(),
      'recordingUrl': recordingUrl,
    };
  }

  static Map<String, dynamic>? _asMap(Object? value) {
    if (value is Map<String, dynamic>) {
      return value;
    }
    if (value is Map) {
      return value.map((key, value) => MapEntry(key.toString(), value));
    }
    return null;
  }

  static List<Map<String, dynamic>> _mapsFrom(Object? value) {
    if (value is! List) {
      return const [];
    }

    final maps = <Map<String, dynamic>>[];
    for (final item in value) {
      final map = _asMap(item);
      if (map != null) {
        maps.add(map);
      }
    }
    return maps;
  }

  static String _readString(Object? value, {String fallback = ''}) {
    final string = value?.toString().trim();
    if (string == null || string.isEmpty) {
      return fallback;
    }
    return string;
  }

  static DateTime _readDateTime(Object? value) {
    final parsed = DateTime.tryParse(value?.toString() ?? '');
    return parsed ?? DateTime.now();
  }

  static Duration _readDuration(Object? value) {
    if (value is int) {
      return Duration(seconds: value);
    }
    if (value is num) {
      return Duration(seconds: value.round());
    }

    final raw = value?.toString().trim() ?? '';
    if (raw.isEmpty) {
      return Duration.zero;
    }

    final parts = raw.split(':').map(int.tryParse).toList();
    if (parts.length == 2 && parts.every((part) => part != null)) {
      return Duration(hours: parts[0]!, minutes: parts[1]!);
    }
    if (parts.length == 3 && parts.every((part) => part != null)) {
      return Duration(hours: parts[0]!, minutes: parts[1]!, seconds: parts[2]!);
    }

    final minutes = int.tryParse(raw.replaceAll(RegExp(r'[^0-9]'), ''));
    return Duration(minutes: minutes ?? 0);
  }

  static ReportProcessingStatus _readStatus(Object? value) {
    final normalized = value?.toString().trim().toLowerCase() ?? '';
    return switch (normalized) {
      'ready' ||
      'saved' ||
      'needs_review' ||
      'completed' ||
      'complete' => ReportProcessingStatus.ready,
      'error' || 'failed' || 'failure' => ReportProcessingStatus.error,
      _ => ReportProcessingStatus.processing,
    };
  }

  static String _readSummary(
    Map<String, dynamic> json,
    Map<String, dynamic> root,
  ) {
    final flat = root['summary'];
    if (flat is String && flat.trim().isNotEmpty) {
      return flat.trim();
    }

    final sections = _mapsFrom(json['summary']);
    if (sections.isEmpty) {
      return 'Отчет пока обрабатывается. Подробности появятся после AI-анализа.';
    }

    return sections
        .map((section) => _readString(section['text'] ?? section['title']))
        .where((text) => text.isNotEmpty)
        .join('\n\n');
  }

  static List<String> _readTopics(
    Map<String, dynamic> json,
    Map<String, dynamic> root,
  ) {
    final topics = _stringsFrom(root['topics']);
    if (topics.isNotEmpty) {
      return topics;
    }

    final chapters = _mapsFrom(json['chapters'])
        .map((chapter) => _readString(chapter['title']))
        .where((title) => title.isNotEmpty)
        .toList();
    if (chapters.isNotEmpty) {
      return chapters;
    }

    return _mapsFrom(json['summary'])
        .map((section) => _readString(section['title']))
        .where((title) => title.isNotEmpty)
        .toList();
  }

  static List<String> _readTakeaways(
    Map<String, dynamic> json,
    Map<String, dynamic> root,
  ) {
    final takeaways = _stringsFrom(root['takeaways']);
    if (takeaways.isNotEmpty) {
      return takeaways;
    }

    final highlights = _mapsFrom(json['highlights'])
        .map((highlight) => _readString(highlight['text'] ?? highlight['note']))
        .where((text) => text.isNotEmpty)
        .toList();
    if (highlights.isNotEmpty) {
      return highlights;
    }

    return _mapsFrom(json['summary'])
        .skip(1)
        .map((section) => _readString(section['text']))
        .where((text) => text.isNotEmpty)
        .toList();
  }

  static List<String> _stringsFrom(Object? value) {
    if (value is! List) {
      return const [];
    }
    return value
        .map((item) => item.toString().trim())
        .where((item) => item.isNotEmpty)
        .toList();
  }

  static List<SpeakerActivityModel> _markMostActive(
    List<SpeakerActivityModel> activity,
  ) {
    if (activity.isEmpty || activity.any((speaker) => speaker.isMostActive)) {
      return activity;
    }

    final maxPercent = activity
        .map((speaker) => speaker.participationPercent)
        .reduce((a, b) => a > b ? a : b);
    return activity
        .map(
          (speaker) => SpeakerActivityModel(
            speakerName: speaker.speakerName,
            talkTime: speaker.talkTime,
            participationPercent: speaker.participationPercent,
            isMostActive: speaker.participationPercent == maxPercent,
          ),
        )
        .toList();
  }

  static String? _readRecordingUrl({
    required Map<String, dynamic> json,
    required Map<String, dynamic> root,
    required String reportId,
    required String? backendBaseUrl,
  }) {
    var raw = _readString(
      json['recordingUrl'] ??
          json['recordingURL'] ??
          json['recording_url'] ??
          root['recordingUrl'] ??
          root['recording_url'],
    );
    final recording = _asMap(json['recording']);
    raw = _readString(
      recording?['url'] ?? recording?['recordingUrl'],
      fallback: raw,
    );

    if (raw.isEmpty && _readString(json['recordingFile']).isNotEmpty) {
      raw = '/api/reports/$reportId/recording/stream';
    }
    if (raw.isEmpty) {
      return null;
    }

    final uri = Uri.tryParse(raw);
    if (uri != null && uri.hasScheme) {
      return raw;
    }

    final base = Uri.tryParse(backendBaseUrl ?? '');
    if (base == null || !base.hasScheme) {
      return raw;
    }

    final path = raw.startsWith('/') ? raw : '/$raw';
    return base.resolve(path).toString();
  }

  static List<ReportModel> mockReports() {
    final now = DateTime.now();
    return [
      ReportModel(
        id: 'report-ready',
        roomName: 'Product Sync AlemLive',
        startedAt: now.subtract(const Duration(days: 1, hours: 3)),
        duration: const Duration(minutes: 48, seconds: 30),
        status: ReportProcessingStatus.ready,
        summary:
            'Команда согласовала сценарий мобильной комнаты, структуру отчетов и порядок подключения LiveKit после UI-этапа.',
        topics: const [
          'Навигация и роли участников',
          'Состояния записи и AI-обработки',
          'Мобильный UX комнаты',
        ],
        takeaways: const [
          'Room UI остается mock до подключения LiveKit.',
          'Отчеты должны открываться прямо с главного экрана.',
          'AI-вопросы пойдут через backend API на следующем этапе.',
        ],
        actionItems: const [
          ActionItemModel(
            task: 'Подготовить API client для списка отчетов',
            owner: 'Mobile',
            dueDate: 'Пятница',
            status: ActionItemStatus.inProgress,
          ),
          ActionItemModel(
            task: 'Согласовать формат transcript JSON',
            owner: 'Backend',
            dueDate: 'Понедельник',
            status: ActionItemStatus.open,
          ),
          ActionItemModel(
            task: 'Проверить UX вкладок на маленьких экранах',
            owner: 'Design',
            status: ActionItemStatus.done,
          ),
        ],
        speakerActivity: const [
          SpeakerActivityModel(
            speakerName: 'Madi',
            talkTime: Duration(minutes: 18, seconds: 20),
            participationPercent: 38,
            isMostActive: true,
          ),
          SpeakerActivityModel(
            speakerName: 'Алия',
            talkTime: Duration(minutes: 14, seconds: 10),
            participationPercent: 29,
            isMostActive: false,
          ),
          SpeakerActivityModel(
            speakerName: 'Данияр',
            talkTime: Duration(minutes: 10, seconds: 30),
            participationPercent: 22,
            isMostActive: false,
          ),
          SpeakerActivityModel(
            speakerName: 'QA',
            talkTime: Duration(minutes: 5, seconds: 30),
            participationPercent: 11,
            isMostActive: false,
          ),
        ],
        transcript: const [
          TranscriptSegmentModel(
            timecode: '00:01:25',
            speakerName: 'Madi',
            text:
                'Давайте сначала доведем UI комнаты и отчетов до рабочего состояния.',
          ),
          TranscriptSegmentModel(
            timecode: '00:06:40',
            speakerName: 'Алия',
            text:
                'Для отчетов важно сразу показать статус обработки AI и превью записи.',
          ),
          TranscriptSegmentModel(
            timecode: '00:14:05',
            speakerName: 'Данияр',
            text: 'Таймкоды транскрипта позже должны перематывать видео.',
          ),
          TranscriptSegmentModel(
            timecode: '00:32:18',
            speakerName: 'QA',
            text: 'Проверим сценарии ошибки, пустого состояния и обработки.',
          ),
        ],
      ),
      ReportModel(
        id: 'report-processing',
        roomName: 'Design Review',
        startedAt: now.subtract(const Duration(hours: 6, minutes: 20)),
        duration: const Duration(minutes: 31),
        status: ReportProcessingStatus.processing,
        summary:
            'AI еще обрабатывает запись. Ниже показан предварительный mock-отчет.',
        topics: const ['Макеты комнаты', 'Bottom sheets', 'Состояния кнопок'],
        takeaways: const [
          'Оставить синий акцент для активного чата.',
          'Кнопки комнаты должны быть удобны большим пальцем.',
        ],
        actionItems: const [
          ActionItemModel(
            task: 'Добавить polling статуса обработки',
            owner: 'Mobile',
            dueDate: 'Следующий этап',
            status: ActionItemStatus.open,
          ),
        ],
        speakerActivity: const [
          SpeakerActivityModel(
            speakerName: 'Алия',
            talkTime: Duration(minutes: 16),
            participationPercent: 52,
            isMostActive: true,
          ),
          SpeakerActivityModel(
            speakerName: 'Madi',
            talkTime: Duration(minutes: 15),
            participationPercent: 48,
            isMostActive: false,
          ),
        ],
        transcript: const [
          TranscriptSegmentModel(
            timecode: '00:00:45',
            speakerName: 'Алия',
            text: 'Нужно сохранить легкий белый фон и не перегружать карточки.',
          ),
          TranscriptSegmentModel(
            timecode: '00:10:12',
            speakerName: 'Madi',
            text:
                'Сделаем вкладки горизонтальными, чтобы структура была как Read.ai.',
          ),
        ],
      ),
      ReportModel(
        id: 'report-error',
        roomName: 'Backend Handoff',
        startedAt: now.subtract(const Duration(days: 3, hours: 2)),
        duration: const Duration(minutes: 22, seconds: 45),
        status: ReportProcessingStatus.error,
        summary:
            'AI-обработка завершилась ошибкой. UI показывает состояние ошибки и доступные данные записи.',
        topics: const ['LiveKit token', 'Recording API', 'Reports API'],
        takeaways: const [
          'Нужно показать понятную ошибку обработки.',
          'Повторная обработка будет backend-сценарием.',
        ],
        actionItems: const [
          ActionItemModel(
            task: 'Добавить retry processing endpoint',
            owner: 'Backend',
            status: ActionItemStatus.open,
          ),
        ],
        speakerActivity: const [
          SpeakerActivityModel(
            speakerName: 'Данияр',
            talkTime: Duration(minutes: 13, seconds: 20),
            participationPercent: 59,
            isMostActive: true,
          ),
          SpeakerActivityModel(
            speakerName: 'Madi',
            talkTime: Duration(minutes: 9, seconds: 25),
            participationPercent: 41,
            isMostActive: false,
          ),
        ],
        transcript: const [
          TranscriptSegmentModel(
            timecode: '00:02:10',
            speakerName: 'Данияр',
            text: 'Recording API должен вернуть статус и идентификатор отчета.',
          ),
          TranscriptSegmentModel(
            timecode: '00:18:02',
            speakerName: 'Madi',
            text: 'На мобильной стороне покажем ошибку без блокировки экрана.',
          ),
        ],
      ),
    ];
  }
}
