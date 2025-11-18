import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/src/services/api_client.dart';
import 'package:user_app/src/models/user.dart';
import 'package:user_app/src/features/auth/application/auth_service.dart';

// Provider for the repository
final userRepositoryProvider = Provider<UserRepository>((ref) {
  return UserRepository(ref.watch(apiClientProvider));
});

class UserRepository {
  final ApiClient _api;

  UserRepository(this._api);

  // Calls get /users/profile
  Future<User> getUserProfile() async {
    final responseData = await _api.get('/users/profile');
    return User.fromJson(responseData);
  }

  // Calls post /users/register
  Future<User> registerUser({
    required String displayName,
    String? profileImageUrl, //optional
  }) async {
    final data = {
      'display_name': displayName,
      'profile_image_url': profileImageUrl ?? '',
    };
    final responseData = await _api.post('/users/register', data);
    return User.fromJson(responseData);
  }
}

final userProfileProvider = FutureProvider<User>((ref) async {
  // check if a user is logged in at all
  final firebaseUser = ref.watch(authStateChangesProvider).value;
  if (firebaseUser == null) {
    throw Exception('User not logged in');
  }

  // User is logged in, try to fetch their profile
  return ref.watch(userRepositoryProvider).getUserProfile();
});
