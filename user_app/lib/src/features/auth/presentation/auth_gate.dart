import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/src/features/auth/application/auth_service.dart';
import 'package:user_app/src/features/auth/application/user_repository.dart';
import 'package:user_app/src/features/auth/presentation/login_screen.dart';
import 'package:user_app/src/features/auth/presentation/profile_creation_screen.dart';
import 'package:user_app/src/features/core/presentation/home_screen.dart';
import 'package:user_app/src/services/api_client.dart';

class AuthGate extends ConsumerWidget {
  const AuthGate({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final authState = ref.watch(authStateChangesProvider);

    return authState.when(
      data: (firebaseUser) {
        if (firebaseUser == null) {
          return const LoginScreen();
        }
        final userProfile = ref.watch(userProfileProvider);

        return userProfile.when(
          data: (sageUser) {
            return const HomeScreen();
          },
          loading: () =>
              const Scaffold(body: Center(child: CircularProgressIndicator())),
          error: (error, stack) {

            if (error is ProfileNotFoundException) {
              return const ProfileCreationScreen();
            }

            return Scaffold(body: Center(child: Text('App Error: $error')));
          },
        );
      },
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (error, stack) =>
          Scaffold(body: Center(child: Text('Auth Error: $error'))),
    );
  }
}
