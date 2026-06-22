import 'package:alem_live_mobile/features/rooms/presentation/livekit/livekit_room_state.dart';
import 'package:alem_live_mobile/features/rooms/presentation/widgets/participant_video_tile.dart';
import 'package:flutter/material.dart';

class RoomVideoGrid extends StatelessWidget {
  const RoomVideoGrid({required this.participants, super.key});

  final List<RoomParticipantView> participants;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final isWide = constraints.maxWidth >= 720;
        final crossAxisCount = isWide ? 2 : 1;

        return GridView.builder(
          padding: EdgeInsets.zero,
          itemCount: participants.length,
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: crossAxisCount,
            crossAxisSpacing: 12,
            mainAxisSpacing: 12,
            childAspectRatio: isWide ? 16 / 9 : 16 / 10,
          ),
          itemBuilder: (context, index) {
            return ParticipantVideoTile(participantView: participants[index]);
          },
        );
      },
    );
  }
}
