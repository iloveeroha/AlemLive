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
  });

  factory ReportModel.fromJson(Map<String, dynamic> json) {
    return ReportModel(
      id: json['id'] as String,
      roomName: json['roomName'] as String,
      startedAt: DateTime.parse(json['startedAt'] as String),
      duration: Duration(seconds: json['durationSeconds'] as int),
      status: ReportProcessingStatus.values.firstWhere(
        (status) => status.name == json['status'],
        orElse: () => ReportProcessingStatus.processing,
      ),
      summary: json['summary'] as String,
      topics: (json['topics'] as List<dynamic>).cast<String>(),
      takeaways: (json['takeaways'] as List<dynamic>).cast<String>(),
      actionItems: (json['actionItems'] as List<dynamic>)
          .map((item) => ActionItemModel.fromJson(item as Map<String, dynamic>))
          .toList(),
      speakerActivity: (json['speakerActivity'] as List<dynamic>)
          .map(
            (item) =>
                SpeakerActivityModel.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
      transcript: (json['transcript'] as List<dynamic>)
          .map(
            (item) =>
                TranscriptSegmentModel.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
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
    };
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
