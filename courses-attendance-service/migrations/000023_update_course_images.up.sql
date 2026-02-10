-- Update courses with course images
-- Using Unsplash for high-quality course images

-- Mathematics courses
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1635070041078-e363dbe005cb?w=800&q=80' WHERE id = 'c0000001-0000-0000-0000-000000000001'; -- Advanced Calculus I (math formulas)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1509228468518-180dd4864904?w=800&q=80' WHERE id = 'c0000002-0000-0000-0000-000000000002'; -- Linear Algebra (matrix/grid)

-- Computer Science courses
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1516116216624-53e697fedbea?w=800&q=80' WHERE id = 'c0000003-0000-0000-0000-000000000003'; -- Data Structures (code/algorithms)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=800&q=80' WHERE id = 'c0000004-0000-0000-0000-000000000004'; -- Algorithms (code on screen)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1526379095098-d400fd0bf935?w=800&q=80' WHERE id = 'c0000005-0000-0000-0000-000000000005'; -- Python Programming (python code)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1547658719-da2b51169166?w=800&q=80' WHERE id = 'c0000006-0000-0000-0000-000000000006'; -- Web Development (laptop/code)

-- Physics courses
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1636466497217-26a8cbeaf0aa?w=800&q=80' WHERE id = 'c0000007-0000-0000-0000-000000000007'; -- Quantum Mechanics (quantum physics)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&q=80' WHERE id = 'c0000008-0000-0000-0000-000000000008'; -- Thermodynamics (space/energy)

-- Business courses
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800&q=80' WHERE id = 'c0000009-0000-0000-0000-000000000009'; -- Business Analytics (charts/graphs)
UPDATE courses SET course_image = 'https://images.unsplash.com/photo-1553729459-efe14ef6055d?w=800&q=80' WHERE id = 'c0000010-0000-0000-0000-000000000010'; -- Financial Accounting (calculator/finance)
