import 'package:dio/dio.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/src/features/auth/application/auth_service.dart';

//custom 404 exception
class ProfileNotFoundException implements Exception {
  final String message = "User profile not found (404).";
  @override
  String toString() => message;
}

// The Riverpod Provider for API Client
final apiClientProvider = Provider<ApiClient>((ref) {
  final baseUrl = 'http://localhost:8080';

  final dio = Dio(BaseOptions(baseUrl: baseUrl));

  // Add Auth Interceptor
  dio.interceptors.add(AuthInterceptor(ref.watch(firebaseAuthProvider)));
  return ApiClient(dio);
});

// API Client Class
class ApiClient {
  final Dio _dio;

  ApiClient(this._dio);

  // Helper for get requests
  Future<Map<String, dynamic>> get(String path) async {
    try {
      final response = await _dio.get(path);
      return response.data;
    } on DioException catch (e) {
      // If we get a 404, throw custom exception
      if (e.response?.statusCode == 404) {
        throw ProfileNotFoundException();
      }
      throw e;
    }
  }

  // Helper for post requests
  Future<Map<String, dynamic>> post(
    String path,
    Map<String, dynamic> data,
  ) async {
    try {
      final response = await _dio.post(path, data: data);
      return response.data;
    } on DioException catch (e) {
      throw e;
    }
  }
}

class AuthInterceptor extends Interceptor {
  final FirebaseAuth _auth;

  AuthInterceptor(this._auth);

  @override
  Future<void> onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final user = _auth.currentUser;
    if (user == null) {
      return handler.next(options);
    }

    try {
      final token = await user.getIdToken();
      options.headers['Authorization'] = 'Bearer $token';
      options.headers['X-Firebase-ID'] = user.uid;

      return handler.next(options);
    } catch (e) {
      return handler.reject(DioException(requestOptions: options, error: e));
    }
  }
}
