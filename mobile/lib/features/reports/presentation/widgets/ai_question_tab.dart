import 'package:alem_live_mobile/app/theme.dart';
import 'package:alem_live_mobile/core/widgets/primary_button.dart';
import 'package:alem_live_mobile/features/ai/domain/usecases/ai_usecases.dart';
import 'package:alem_live_mobile/features/reports/domain/entities/report.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AiQuestionTab extends ConsumerStatefulWidget {
  const AiQuestionTab({required this.report, super.key});

  final Report report;

  @override
  ConsumerState<AiQuestionTab> createState() => _AiQuestionTabState();
}

class _AiQuestionTabState extends ConsumerState<AiQuestionTab> {
  final _questionController = TextEditingController();
  final List<_AiThreadItem> _items = [];
  bool _isSubmitting = false;

  @override
  void dispose() {
    _questionController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        TextField(
          controller: _questionController,
          minLines: 2,
          maxLines: 4,
          decoration: const InputDecoration(
            labelText: 'Вопрос по встрече',
            hintText: 'Например: какие задачи взял backend?',
            prefixIcon: Icon(Icons.auto_awesome_rounded),
          ),
        ),
        const SizedBox(height: 12),
        PrimaryButton(
          label: 'Спросить',
          icon: Icons.send_rounded,
          isLoading: _isSubmitting,
          onPressed: _ask,
        ),
        const SizedBox(height: 18),
        if (_items.isEmpty)
          const _AiAnswerCard(
            question: 'Что можно спросить?',
            answer:
                'Спросите про решения встречи, задачи, сроки, активность участников или конкретные моменты из транскрипта.',
          )
        else
          ..._items.map(
            (item) => Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _AiAnswerCard(
                question: item.question,
                answer: item.answer,
              ),
            ),
          ),
      ],
    );
  }

  Future<void> _ask() async {
    final question = _questionController.text.trim();
    if (question.isEmpty || _isSubmitting) {
      return;
    }

    setState(() => _isSubmitting = true);
    try {
      // TODO: Replace mock fallback with ask AI API response when backend is ready.
      final answer = await ref
          .read(askAiQuestionUseCaseProvider)
          .call(
            reportId: widget.report.id,
            roomName: widget.report.roomName,
            question: question,
            fallbackTakeaway: widget.report.takeaways.first,
          );
      setState(() {
        _items.insert(0, _AiThreadItem(question: question, answer: answer));
        _questionController.clear();
      });
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    } finally {
      if (mounted) {
        setState(() => _isSubmitting = false);
      }
    }
  }
}

class _AiThreadItem {
  const _AiThreadItem({required this.question, required this.answer});

  final String question;
  final String answer;
}

class _AiAnswerCard extends StatelessWidget {
  const _AiAnswerCard({required this.question, required this.answer});

  final String question;
  final String answer;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppTheme.border),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(question, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          Text(answer),
        ],
      ),
    );
  }
}
