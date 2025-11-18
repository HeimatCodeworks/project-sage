import 'package:flutter/material.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Project Sage Home'),
      ),
      body: const Center(child: Text('Home Screen (User is Logged In)')),
    );
  }
}
