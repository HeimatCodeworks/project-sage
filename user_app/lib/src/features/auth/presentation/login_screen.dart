import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/src/common_widgets/primary_button.dart';
import 'package:user_app/src/features/auth/application/auth_service.dart';
import 'package:user_app/src/features/auth/presentation/sign_up_screen.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isLoading = false;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _submitLogin() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() { _isLoading = true; });
    try {
      final email = _emailController.text;
      final password = _passwordController.text;
      await ref.read(authServiceProvider).signInWithEmailPassword(email, password);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to sign in: $e')),
        );
      }
    }
    if (mounted) {
      setState(() { _isLoading = false; });
    }
  }

  final _textStyle = const TextStyle(fontSize: 18);
  final _hintStyle = const TextStyle(fontSize: 18, color: Colors.black54);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Sign In')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24.0),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const SizedBox(height: 20),
              Text(
                'Welcome to Project Sage',
                style: _textStyle.copyWith(
                  fontSize: 24,
                  fontWeight: FontWeight.w600,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 40),
              TextFormField(
                controller: _emailController,
                style: _textStyle,
                decoration: InputDecoration(
                  labelText: 'Email',
                  hintText: 'you@example.com',
                  labelStyle: _textStyle,
                  hintStyle: _hintStyle,
                  border: const OutlineInputBorder(),
                ),
                keyboardType: TextInputType.emailAddress,
                validator: (value) => (value == null || !value.contains('@'))
                    ? 'Please enter a valid email'
                    : null,
              ),
              const SizedBox(height: 20),
              TextFormField(
                controller: _passwordController,
                style: _textStyle,
                decoration: InputDecoration(
                  labelText: 'Password',
                  labelStyle: _textStyle,
                  hintStyle: _hintStyle,
                  border: const OutlineInputBorder(),
                ),
                obscureText: true,
                validator: (value) => (value == null || value.length < 6)
                    ? 'Password must be at least 6 characters'
                    : null,
              ),
              const SizedBox(height: 30),
              PrimaryButton(
                text: 'Sign In',
                isLoading: _isLoading,
                onPressed: _isLoading ? null : _submitLogin,
              ),

              const SizedBox(height: 20),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(
                    "Don't have an account?",
                    style: _textStyle.copyWith(fontSize: 16),
                  ),
                  TextButton(
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (context) => const SignUpScreen(),
                        ),
                      );
                    },
                    child: Text(
                      'Sign Up',
                      style: _textStyle.copyWith(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
