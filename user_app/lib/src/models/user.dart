import 'package:equatable/equatable.dart';

class User extends Equatable {
  const User({
    required this.userId,
    required this.displayName,
    required this.profileImageUrl,
    required this.membershipTier,
    required this.assistanceTokenBalance,
    required this.role,
  });

  final String userId;
  final String displayName;
  final String profileImageUrl;
  final String membershipTier;
  final int assistanceTokenBalance;
  final String role;

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      userId: json['user_id'] as String,
      displayName: json['display_name'] as String,
      profileImageUrl: json['profile_image_url'] as String,
      membershipTier: json['membership_tier'] as String,
      assistanceTokenBalance: json['assistance_token_balance'] as int,
      role: json['role'] as String,
    );
  }

  @override
  List<Object?> get props => [
    userId,
    displayName,
    profileImageUrl,
    membershipTier,
    assistanceTokenBalance,
    role,
  ];
}
