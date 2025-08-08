-- Create user_exams table
CREATE TABLE IF NOT EXISTS user_exams (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'assigned' CHECK (status IN ('assigned', 'started', 'completed', 'expired')),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, exam_id)
);

-- Create results table
CREATE TABLE IF NOT EXISTS results (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    exam_id INTEGER NOT NULL REFERENCES exams(id) ON DELETE CASCADE,
    user_exam_id INTEGER NOT NULL REFERENCES user_exams(id) ON DELETE CASCADE,
    score DECIMAL(5,2) NOT NULL,
    total_points INTEGER NOT NULL,
    max_points INTEGER NOT NULL,
    passed BOOLEAN DEFAULT false,
    answers JSONB,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(user_exam_id)
);

-- Create indexes for user_exams
CREATE INDEX IF NOT EXISTS idx_user_exams_user_id ON user_exams(user_id);
CREATE INDEX IF NOT EXISTS idx_user_exams_exam_id ON user_exams(exam_id);
CREATE INDEX IF NOT EXISTS idx_user_exams_status ON user_exams(status);
CREATE INDEX IF NOT EXISTS idx_user_exams_started_at ON user_exams(started_at);
CREATE INDEX IF NOT EXISTS idx_user_exams_expires_at ON user_exams(expires_at);

-- Create indexes for results
CREATE INDEX IF NOT EXISTS idx_results_user_id ON results(user_id);
CREATE INDEX IF NOT EXISTS idx_results_exam_id ON results(exam_id);
CREATE INDEX IF NOT EXISTS idx_results_user_exam_id ON results(user_exam_id);
CREATE INDEX IF NOT EXISTS idx_results_score ON results(score);
CREATE INDEX IF NOT EXISTS idx_results_passed ON results(passed);
CREATE INDEX IF NOT EXISTS idx_results_created_at ON results(created_at);
CREATE INDEX IF NOT EXISTS idx_results_deleted_at ON results(deleted_at);

