-- Add role and subscription_type columns to user table
ALTER TABLE "user" ADD COLUMN role VARCHAR(20) DEFAULT 'linko-user';
ALTER TABLE "user" ADD COLUMN subscription_type VARCHAR(20) DEFAULT 'free-tier';

-- Update existing admin users
UPDATE "user" SET role = 'admin' WHERE is_admin = true;

-- Create an index on role for faster lookup
CREATE INDEX idx_user_role ON "user" (role);

-- Create a function to ensure subscription_type is valid for linko-users
CREATE OR REPLACE FUNCTION check_subscription_type()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role = 'linko-user' AND 
       NEW.subscription_type NOT IN ('free-tier', 'linko-plus', 'linko-vip') THEN
        NEW.subscription_type := 'free-tier';
    END IF;
    
    -- Admin users don't need subscription
    IF NEW.role = 'admin' THEN
        NEW.subscription_type := NULL;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create a trigger to enforce subscription type constraints
CREATE TRIGGER enforce_subscription_type
BEFORE INSERT OR UPDATE ON "user"
FOR EACH ROW
EXECUTE FUNCTION check_subscription_type();
