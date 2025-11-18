import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:user_app/src/common_widgets/primary_button.dart';
import 'package:user_app/src/features/auth/application/user_repository.dart';

class ProfileCreationScreen extends ConsumerStatefulWidget {
  const ProfileCreationScreen({super.key});

  @override
  ConsumerState<ProfileCreationScreen> createState() =>
      _ProfileCreationScreenState();
}

class _ProfileCreationScreenState extends ConsumerState<ProfileCreationScreen> {
  final _nameController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isLoading = false;

  @override
  void dispose() {
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _submitProfile() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }
    setState(() {
      _isLoading = true;
    });

    try {
      final displayName = _nameController.text;

      // Call the repository to register the user
      await ref
          .read(userRepositoryProvider)
          .registerUser(
            displayName: displayName,
          );

      ref.invalidate(userProfileProvider);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Failed to create profile: $e')));
      }
      setState(() {
        _isLoading = false;
      });
    }
  }

  final _textStyle = const TextStyle(fontSize: 18);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Create Your Profile')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24.0),
        child: Form(
          key: _formKey,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const SizedBox(height: 20),
              Text(
                'Welcome! One last step.',
                style: _textStyle.copyWith(
                  fontSize: 24,
                  fontWeight: FontWeight.w600,
                ),
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 10),
              Text(
                'Please enter your display name.',
                style: _textStyle,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 40),
              TextFormField(
                controller: _nameController,
                style: _textStyle,
                decoration: InputDecoration(
                  labelText: 'Display Name',
                  labelStyle: _textStyle,
                  border: const OutlineInputBorder(),
                ),
                validator: (value) => (value == null || value.isEmpty)
                    ? 'Please enter a name'
                    : null,
              ),
              const SizedBox(height: 30),
              // TODO add upload photo button
              PrimaryButton(
                text: 'Save and Continue',
                isLoading: _isLoading,
                onPressed: _isLoading ? null : _submitProfile,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
